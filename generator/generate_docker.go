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

type GenerateDocker struct {
	BaseGenerator
	dockerCompose *DockerCompose
}
type DockerCompose struct {
	Version  string                    `yaml:"version"`
	Services map[string]*DockerService `yaml:"services"`
}
type BuildService struct {
	Context    string `yaml:"context"`
	DockerFile string `yaml:"dockerfile"`
}
type DockerService struct {
	Build         BuildService `yaml:"build"`
	Restart       string       `yaml:"restart"`
	Volumes       []string
	ContainerName string   `yaml:"container_name"`
	Ports         []string `yaml:"ports"`
}

func NewGenerateDocker() Gen {
	i := &GenerateDocker{}
	i.dockerCompose = &DockerCompose{}
	i.dockerCompose.Version = "2"
	i.dockerCompose.Services = map[string]*DockerService{}
	i.fs = fs.Get()
	return i
}

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
	pth, err := utils.GetDockerFileProjecPath()
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

ENTRYPOINT  go install %s/%s/cmd && /go/bin/cmd
`
	fpath := "/go/src/" + pth
	err = g.addToDockerCompose(name, fpath, httpFilePath, grpcFilePath)
	if err != nil {
		return err
	}

	dockerFile = fmt.Sprintf(dockerFile, fpath, fpath, pth, name)
	return g.fs.WriteFile(path.Join(name, "Dockerfile"), dockerFile, true)
}

func (g *GenerateDocker) addToDockerCompose(name, pth, httpFilePath, grpcFilePath string) (err error) {
	hasHttp := false
	hasGRPC := false
	if b, err := g.fs.Exists(httpFilePath); err != nil {
		return err
	} else if b {
		hasHttp = true
	}
	if b, err := g.fs.Exists(grpcFilePath); err != nil {
		return err
	} else if b {
		hasGRPC = true
	}
	usedPorts := []string{}
	for _, v := range g.dockerCompose.Services {
		for _, p := range v.Ports {
			pt := strings.Split(p, ":")
			usedPorts = append(usedPorts, pt[0])
		}
	}
	if g.dockerCompose.Services[name] == nil {
		g.dockerCompose.Services[name] = &DockerService{
			Build: BuildService{
				Context:    ".",
				DockerFile: name + "/Dockerfile",
			},
			Restart: "always",
			Volumes: []string{
				".:" + pth,
			},
		}
		if hasHttp {
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
			g.dockerCompose.Services[name].Ports = []string{
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
			if g.dockerCompose.Services[name].Ports == nil {
				g.dockerCompose.Services[name].Ports = []string{}
			}
			g.dockerCompose.Services[name].Ports = append(
				g.dockerCompose.Services[name].Ports,
				fmt.Sprintf("%d", grpcExpose)+":8082",
			)
			usedPorts = append(usedPorts, fmt.Sprintf("%d", grpcExpose))
		}
	}
	return
}
