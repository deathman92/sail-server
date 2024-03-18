package main

import (
	"log"
	"net/http"
	"regexp"
	"strings"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
)

var availableServices = []string{
	"mysql",
	"pgsql",
	"mariadb",
	"redis",
	"memcached",
	"meilisearch",
	"typesense",
	"minio",
	"mailpit",
	"selenium",
	"soketi",
}

var availablePhps = []string{"74", "80", "81", "82", "83"}

type params struct {
	Name string   `validate:"alphadash"`
	Php  string   `validate:"phpversion"`
	With []string `validate:"gt=0,services"`
}

type scriptParams struct {
	Name         string
	Php          string
	With         string
	Services     string
	Database     string
	Pest         string
	Devcontainer string
}

var validate *validator.Validate
var aplhadashRegex *regexp.Regexp

func main() {
	r := gin.Default()
	r.GET("/", func(ctx *gin.Context) {
		ctx.Redirect(http.StatusFound, "https://laravel.com/docs")
	})

	validate = validator.New()
	validate.RegisterValidation("alphadash", validateAlphaDash)
	validate.RegisterValidation("phpversion", validatePhpVersion)
	validate.RegisterValidation("services", validateServices)
	aplhadashRegex = regexp.MustCompile(`\A[\pL\pM\pN_-]+\z`)

	scriptTemplate, err := template.ParseFiles("./script.sh")
	if err != nil {
		log.Fatal(err)
	}

	r.GET("/:name", func(ctx *gin.Context) {
		name := ctx.Param("name")
		php := ctx.DefaultQuery("php", "83")
		with := lo.Uniq(strings.Split(ctx.DefaultQuery("with", "mysql,redis,meilisearch,mailpit,selenium"), ","))

		p := params{
			Name: name,
			Php:  php,
			With: with,
		}

		err := validate.Struct(p)
		if err != nil {
			errs := err.(validator.ValidationErrors)

			if lo.ContainsBy(errs, func(item validator.FieldError) bool { return item.Field() == "Name" }) {
				ctx.String(http.StatusBadRequest, "Invalid site name. Please only use alpha-numeric characters, dashes, and underscores.")
				return
			}

			if lo.ContainsBy(errs, func(item validator.FieldError) bool { return item.Field() == "Php" }) {
				ctx.String(http.StatusBadRequest, "Invalid PHP version. Please specify a supported version (74, 80, 81, 82 or 83).")
				return
			}

			if lo.ContainsBy(errs, func(item validator.FieldError) bool { return item.Field() == "With" }) {
				ctx.String(http.StatusBadRequest, "Invalid service name. Please provide one or more of the supported services (%v) or \"none\".", strings.Join(availableServices, ", "))
				return
			}
		}

		database := "--database mysql"
		if lo.Contains(p.With, "pgsql") {
			database = "--database pgsql"
		} else if lo.Contains(p.With, "mariadb") {
			database = "--database mariadb"
		} else if lo.Contains(p.With, "none") {
			database = ""
		}

		pest := ""
		if ctx.Request.URL.Query().Has("pest") {
			pest = "--pest"
		}
		devcontainer := ""
		if ctx.Request.URL.Query().Has("devcontainer") {
			devcontainer = "--devcontainer"
		}

		scriptTemplate.Execute(ctx.Writer, scriptParams{
			Name:         p.Name,
			Php:          p.Php,
			With:         strings.Join(p.With, ","),
			Services:     strings.Join(p.With, " "),
			Database:     database,
			Pest:         pest,
			Devcontainer: devcontainer,
		})
		ctx.Status(http.StatusOK)
	})
	r.Run()
}

func validateAlphaDash(fl validator.FieldLevel) bool {
	return aplhadashRegex.Match([]byte(fl.Field().String()))
}

func validatePhpVersion(fl validator.FieldLevel) bool {
	return lo.Contains(availablePhps, fl.Field().String())
}

func validateServices(fl validator.FieldLevel) bool {
	if fl.Field().Len() == 1 {
		return fl.Field().Index(0).String() == "none"
	}
	for i := 0; i < fl.Field().Len(); i++ {
		if !lo.Contains(availableServices, fl.Field().Index(i).String()) {
			return false
		}
	}
	return true
}
