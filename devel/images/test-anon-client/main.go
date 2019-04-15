package main

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	anonConfig := rest.AnonymousClientConfig(config)
	clientset, err := kubernetes.NewForConfig(anonConfig)
	if err != nil {
		panic(err)
	}
	secrets, err :=
		clientset.CoreV1().Secrets("default").List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("secrets: %#+v\n", secrets)
}
