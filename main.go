package main

import "kit/watch"

func main() {
	//viper.Set("testFs", afero.NewBasePathFs(fs.AppFs(), "test"))
	//viper.Set("testPath", "test")
	//start := time.Now()
	//s, _ := service.Read("abc")
	//s.Generate()
	//end := time.Now()
	//fmt.Println(end.Sub(start).Seconds())
	//matches, err := zglob.Glob("**/*")
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//fmt.Println(matches)
	watch.Run()
}
