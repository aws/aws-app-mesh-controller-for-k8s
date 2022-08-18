package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	host, port, namespace := os.Getenv("HOST"), os.Getenv("PORT"), os.Getenv("NAMESPACE")

	log.Printf("HOST: %s", host)
	log.Printf("PORT: %s", port)
	log.Printf("NAMESPACE: %s", namespace)

	http.HandleFunc("/ping", func(w http.ResponseWriter, req *http.Request) {})
	http.HandleFunc("/color", func(w http.ResponseWriter, req *http.Request) {
		resp, err := http.Get(fmt.Sprintf("http://%s", host))
		if err != nil {
			http.Error(w, err.Error(), 500)
			log.Printf("Could not get color: %v", err)
			return
		}

		defer resp.Body.Close()

		color, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			log.Printf("Could not read response body: %v", err)
			return
		}

		fmt.Fprint(w, string(color))
	})

	resp, err := http.Get(fmt.Sprintf("http://%s", host))
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	defer resp.Body.Close()

	color, _ := ioutil.ReadAll(resp.Body)
	log.Println(string(color))

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("error loading kubernetes config: %s", err)
		os.Exit(1)
	}

	k8scli, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("error creating kubernetes client: %s", err)
		os.Exit(1)
	}

	pods, err := k8scli.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("error getting pods with namespace %s: %s", namespace, err)
		os.Exit(1)
	}

	// annotate with color
	for _, pod := range pods.Items {
		newPod := pod.DeepCopy()

		ann := newPod.ObjectMeta.Annotations
		if ann == nil {
			ann = make(map[string]string)
		}

		ann["color"] = string(color)
		newPod.ObjectMeta.Annotations = ann

		_, err := k8scli.CoreV1().Pods(newPod.ObjectMeta.Namespace).Update(context.TODO(), newPod, metav1.UpdateOptions{})
		if err != nil {
			log.Printf("error updating pod with new annotation: %s", err)
		}
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", port), nil))
}
