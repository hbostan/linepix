package linepix

import (
	"image"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"log"
	"os"

	"github.com/nfnt/resize"
)

func ReadImage(imageName string) image.Image {
	imgHandle, err := os.Open(imageName)
	if err != nil {
		log.Println("⚠ Cannot open input image.", err)
		return nil
	}
	img, ext, err := image.Decode(imgHandle)
	if err != nil {
		log.Println("⚠ Couldn't decode the image. Make sure it's a JPEG or PNG image. 🖼", err)
		return nil
	}
	log.Printf("👨🏻‍💻 Image Decoded (%v)\n", ext)
	return img
}

func SaveImage(fileName string, img image.Image) error {
	handle, err := os.Create(fileName)
	if err != nil {
		log.Println("⚠ Can't create file to save image.", err)
		return err
	}
	png.Encode(handle, img)
	return nil
}

func ResizeImage(img image.Image, width, height uint) image.Image {
	return resize.Resize(width, height, img, resize.Lanczos3)
}
