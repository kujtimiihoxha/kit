package generator

import (
	"fmt"
	"path"

	yaml "gopkg.in/yaml.v2"

	"strings"

	"github.com/kujtimiihoxha/kit/fs"
	"github.com/kujtimiihoxha/kit/utils"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// GenerateDocker implements Gen and is used to generate
// docker files for services.
type GenerateDocker struct {
	BaseGenerator
	dockerCompose *DockerCompose
	glide         bool
}

// DockerCompose represents the docker-compose.yml
type DockerCompose struct {
	Version  string                 `yaml:"version"`
	Services map[string]interface{} `yaml:"services"`
}

// BuildService represents one docker service build.
type BuildService struct {
	Context    string `yaml:"context"`
	DockerFile string `yaml:"dockerfile"`
}

// DockerService represents one docker service.
type DockerService struct {
	Build         BuildService `yaml:"build"`
	Restart       string       `yaml:"restart"`
	Volumes       []string
	ContainerName string   `yaml:"container_name"`
	Ports         []string `yaml:"ports"`
}

// NewGenerateDocker returns a new docker generator.
func NewGenerateDocker(glide bool) Gen {
	i := &GenerateDocker{
		glide: glide,
	}
	i.dockerCompose = &DockerCompose{}
	i.dockerCompose.Version = "2"
	i.dockerCompose.Services = map[string]interface{}{}
	i.fs = fs.Get()
	return i
}

// Generate generates the docker configurations.
func (g *GenerateDocker) Generate() (err error) {
	f, err := g.fs.Fs.Open(".")
	if err != nil {
		return err
	}
	names, err := f.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, v := range names {
		if b, err := afero.IsDir(g.fs.Fs, v); err != nil {
			return err
		} else if !b {
			continue
		}
		svcFilePath := path.Join(
			fmt.Sprintf(viper.GetString("gk_service_path_format"), utils.ToLowerSnakeCase(v)),
			viper.GetString("gk_service_file_name"),
		)
		httpFilePath := path.Join(
			fmt.Sprintf(viper.GetString("gk_http_path_format"), utils.ToLowerSnakeCase(v)),
			viper.GetString("gk_http_file_name"),
		)
		grpcFilePath := path.Join(
			fmt.Sprintf(viper.GetString("gk_grpc_path_format"), utils.ToLowerSnakeCase(v)),
			viper.GetString("gk_grpc_file_name"),
		)
		err = g.generateDockerFile(v, svcFilePath, httpFilePath, grpcFilePath)
		if err != nil {
			return err
		}
	}
	d, err := yaml.Marshal(g.dockerCompose)
	if err != nil {
		return err
	}
	return g.fs.WriteFile("docker-compose.yml", string(d), true)
}
func (g *GenerateDocker) generateDockerFile(name, svcFilePath, httpFilePath, grpcFilePath string) (err error) {
	pth, err := utils.GetDockerFileProjectPath()
	if err != nil {
		return err
	}
	if b, err := g.fs.Exists(path.Join(name, "Dockerfile")); err != nil {
		return err
	} else if b {
		pth = "/go/src/" + pth
		return g.addToDockerCompose(name, pth, httpFilePath, grpcFilePath)
	}
	if b, err := g.fs.Exists("docker-compose.yml"); err != nil {
		return err
	} else if b {
		r, err := g.fs.ReadFile("docker-compose.yml")
		if err != nil {
			return err
		}
		err = yaml.Unmarshal([]byte(r), g.dockerCompose)
		if err != nil {
			return err
		}
	}
	isService := false
	if b, err := g.fs.Exists(svcFilePath); err != nil {
		return err
	} else if b {
		isService = true
	}

	if !isService {
		return
	}
	dockerFile := `FROM golang

RUN mkdir -p %s

ADD . %s

RUN go get  -t -v ./...
RUN go get  github.com/canthefason/go-watcher
RUN go install github.com/canthefason/go-watcher/cmd/watcher

ENTRYPOINT  watcher -run %s/%s/cmd  -watch %s/%s
`
	if g.glide {
		dockerFile = `FROM golang

RUN mkdir -p %s

ADD . %s

RUN curl https://glide.sh/get | sh
RUN go get  github.com/canthefason/go-watcher
RUN go install github.com/canthefason/go-watcher/cmd/watcher

RUN cd %s && glide install

ENTRYPOINT  watcher -run %s/%s/cmd -watch %s/%s
`
	}
	fpath := "/go/src/" + pth
	err = g.addToDockerCompose(name, fpath, httpFilePath, grpcFilePath)
	if err != nil {
		return err
	}
	if g.glide {
		dockerFile = fmt.Sprintf(dockerFile, fpath, fpath, fpath, pth, name, pth, name)

	} else {
		dockerFile = fmt.Sprintf(dockerFile, fpath, fpath, pth, name, pth, name)
	}
	return g.fs.WriteFile(path.Join(name, "Dockerfile"), dockerFile, true)
}

func (g *GenerateDocker) addToDockerCompose(name, pth, httpFilePath, grpcFilePath string) (err error) {
	hasHTTP := false
	hasGRPC := false
	if b, err := g.fs.Exists(httpFilePath); err != nil {
		return err
	} else if b {
		hasHTTP = true
	}
	if b, err := g.fs.Exists(grpcFilePath); err != nil {
		return err
	} else if b {
		hasGRPC = true
	}
	usedPorts := []string{}
	for _, v := range g.dockerCompose.Services {
		k, ok := v.(map[interface{}]interface{})
		if ok {
			for _, p := range k["ports"].([]interface{}) {
				pt := strings.Split(p.(string), ":")
				usedPorts = append(usedPorts, pt[0])
			}
		} else {
			for _, p := range v.(*DockerService).Ports {
				pt := strings.Split(p, ":")
				usedPorts = append(usedPorts, pt[0])
			}
		}

	}
	if g.dockerCompose.Services[name] == nil {
		g.dockerCompose.Services[name] = &DockerService{
			Build: BuildService{
				Context:    ".",
				DockerFile: name + "/Dockerfile",
			},
			Restart:       "always",
			ContainerName: name,
			Volumes: []string{
				".:" + pth,
			},
		}
		if hasHTTP {
			httpExpose := 8800
			for {
				ex := false
				for _, v := range usedPorts {
					if v == fmt.Sprintf("%d", httpExpose) {
						ex = true
						break
					}
				}
				if ex {
					httpExpose++
				} else {
					break
				}
			}
			g.dockerCompose.Services[name].(*DockerService).Ports = []string{
				fmt.Sprintf("%d", httpExpose) + ":8081",
			}
			usedPorts = append(usedPorts, fmt.Sprintf("%d", httpExpose))
		}
		if hasGRPC {
			grpcExpose := 8800
			for {
				ex := false
				for _, v := range usedPorts {
					if v == fmt.Sprintf("%d", grpcExpose) {
						ex = true
						break
					}
				}
				if ex {
					grpcExpose++
				} else {
					break
				}
			}
			if g.dockerCompose.Services[name].(*DockerService).Ports == nil {
				g.dockerCompose.Services[name].(*DockerService).Ports = []string{}
			}
			g.dockerCompose.Services[name].(*DockerService).Ports = append(
				g.dockerCompose.Services[name].(*DockerService).Ports,
				fmt.Sprintf("%d", grpcExpose)+":8082",
			)
		}
	}
	return
}
