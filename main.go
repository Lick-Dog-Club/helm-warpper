package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

type HelmConfig struct {
	UploadPath string        `yaml:"uploadPath"`
	HelmRepos  []*repo.Entry `yaml:"helmRepos"`
}

var (
	tokenFile  = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	rootCAFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

var (
	settings          = cli.New()
	defaultUploadPath = "/tmp/charts"
	helmConfig        = &HelmConfig{}
)

func main() {
	var (
		listenHost string
		listenPort string
		config     string
	)

	flag.Set("logtostderr", "true")
	pflag.CommandLine.StringVar(&listenHost, "addr", "0.0.0.0", "server listen addr")
	pflag.CommandLine.StringVar(&listenPort, "port", "8080", "server listen port")
	pflag.CommandLine.StringVar(&config, "config", "config.yaml", "helm wrapper config")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	settings.AddFlags(pflag.CommandLine)
	pflag.Parse()
	defer glog.Flush()

	setInK8s(settings)
	configBody, err := ioutil.ReadFile(config)
	if err != nil {
		glog.Fatalln(err)
	}
	err = yaml.Unmarshal(configBody, helmConfig)
	if err != nil {
		glog.Fatalln(err)
	}

	// upload chart path
	if helmConfig.UploadPath == "" {
		helmConfig.UploadPath = defaultUploadPath
	} else {
		if !filepath.IsAbs(helmConfig.UploadPath) {
			glog.Fatalln("charts upload path is not absolute")
		}
	}
	_, err = os.Stat(helmConfig.UploadPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(helmConfig.UploadPath, 0755)
			if err != nil {
				glog.Fatalln(err)
			}
		} else {
			glog.Fatalln(err)
		}
	}

	// init repo
	for _, c := range helmConfig.HelmRepos {
		err = initRepository(c)
		if err != nil {
			glog.Fatalln(err)
		}
	}

	// router
	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome helm wrapper server")
	})

	// register router
	RegisterRouter(router)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", listenHost, listenPort),
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			glog.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	glog.Infoln("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func setInK8s(settings *cli.EnvSettings) {
	if exists(tokenFile) && exists(rootCAFile) {
		log.Println("setInK8s")
		host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
		settings.KubeAPIServer = "https://" + net.JoinHostPort(host, port)
		token, _ := ioutil.ReadFile(tokenFile)
		settings.KubeToken = string(token)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
