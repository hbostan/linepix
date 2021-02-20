package main

import "github.com/hbostann/linepix"

func main() {
	image := linepix.ReadImage("test.png")
	image = linepix.ResizeImage(image, 0, 250)
	linepix.SaveImage("output.png", image)

}
