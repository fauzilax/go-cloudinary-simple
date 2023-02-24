package main

import (
	"context"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/cloudinary/cloudinary-go"
	"github.com/cloudinary/cloudinary-go/api/uploader"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func ImageUploadHelper(input interface{}) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//create cloudinary instance
	cld, err := cloudinary.NewFromParams("CLOUDINARY_CLOUD_NAME", "CLOUDINARY_API_KEY", "CLOUDINARY_API_SECRET")
	if err != nil {
		return "", err
	}

	//upload file
	uploadParam, err := cld.Upload.Upload(ctx, input, uploader.UploadParams{Folder: "CLOUDINARY_UPLOAD_FOLDER"})
	if err != nil {
		return "", err
	}
	return uploadParam.SecureURL, nil
}

// --------------Helper Model ----------------
type File struct {
	File multipart.File `json:"file,omitempty" validate:"required"`
}

type Url struct {
	Url string `json:"url,omitempty" validate:"required"`
}

// ----------------Helper Service ------------------
var (
	validate = validator.New()
)

type mediaUpload interface {
	FileUpload(file File) (string, error)
	RemoteUpload(url Url) (string, error)
}
type media struct{}

func NewMediaUpload() mediaUpload {
	return &media{}
}

func (*media) FileUpload(file File) (string, error) {
	//validate
	err := validate.Struct(file)
	if err != nil {
		return "", err
	}

	//upload
	uploadUrl, err := ImageUploadHelper(file.File)
	if err != nil {
		return "", err
	}
	return uploadUrl, nil
}

func (*media) RemoteUpload(url Url) (string, error) {
	//validate
	err := validate.Struct(url)
	if err != nil {
		return "", err
	}

	//upload
	uploadUrl, errUrl := ImageUploadHelper(url.Url)
	if errUrl != nil {
		return "", err
	}
	return uploadUrl, nil
}

// -----------DTOS Helper---------------
type MediaDto struct {
	StatusCode int       `json:"statusCode"`
	Message    string    `json:"message"`
	Data       *echo.Map `json:"data"`
}

func main() {
	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.CORS())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}, error=${error}\n",
	}))

	e.POST("/upload", func(c echo.Context) error {
		//upload
		formHeader, err := c.FormFile("file")
		if err != nil {
			return c.JSON(http.StatusInternalServerError,
				MediaDto{
					StatusCode: http.StatusInternalServerError,
					Message:    "error",
					Data:       &echo.Map{"data": "Select a file to upload"},
				})
		}

		//get file from header
		formFile, err := formHeader.Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError,
				MediaDto{
					StatusCode: http.StatusInternalServerError,
					Message:    "error",
					Data:       &echo.Map{"data": err.Error()},
				})
		}

		uploadUrl, err := NewMediaUpload().FileUpload(File{File: formFile})
		if err != nil {
			return c.JSON(http.StatusInternalServerError,
				MediaDto{
					StatusCode: http.StatusInternalServerError,
					Message:    "error",
					Data:       &echo.Map{"data": err.Error()},
				})
		}

		return c.JSON(http.StatusOK, MediaDto{
			StatusCode: http.StatusOK,
			Message:    "success",
			Data:       &echo.Map{"data": uploadUrl},
		})
	})
	if err := e.Start(":8000"); err != nil {
		log.Println(err.Error())
	}

}
