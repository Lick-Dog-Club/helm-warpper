package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/golang/glog"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func actionConfigInit(namespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	var clientConfig *genericclioptions.ConfigFlags
	if exists(tokenFile) && exists(rootCAFile) {
		log.Println("in k8s actionConfigInit")
		clientConfig = genericclioptions.NewConfigFlags(true)
		clientConfig.Namespace = &namespace
		clientConfig.Context = &settings.KubeContext
		// caFile, _ := ioutil.ReadFile(rootCAFile)
		// strCaFile := string(caFile)
		// clientConfig.CertFile = &strCaFile
		clientConfig.CAFile = &rootCAFile
		token, _ := ioutil.ReadFile(tokenFile)
		strToken := string(token)
		clientConfig.BearerToken = &strToken
	} else {
		clientConfig = kube.GetConfig(settings.KubeConfig, settings.KubeContext, namespace)
	}
	if settings.KubeToken != "" {
		clientConfig.BearerToken = &settings.KubeToken
	}
	if settings.KubeAPIServer != "" {
		clientConfig.APIServer = &settings.KubeAPIServer
	}
	err := actionConfig.Init(clientConfig, namespace, os.Getenv("HELM_DRIVER"), glog.Infof)
	if err != nil {
		glog.Errorf("%+v", err)
		return nil, err
	}

	return actionConfig, nil
}
