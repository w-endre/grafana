package catalog

import (
	"context"
	"errors"
	"log"
	"path/filepath"
	"time"

	"github.com/grafana/grafana/pkg/registry"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const ServiceName = "Catalog"

type Service struct {
	Client kubernetes.Interface
}

func (s *Service) Init() error {
	clientset, err := connect()
	if err != nil {
		return err
	}

	s.Client = clientset
	return nil
}

func init() {
	registry.Register(&registry.Descriptor{
		Name:         ServiceName,
		Instance:     &Service{},
		InitPriority: registry.High,
	})
}

func (s *Service) Run(ctx context.Context) error {
	return s.startServiceInformer()
}

func connect() (*kubernetes.Clientset, error) {
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		return nil, errors.New("could not get filepath of kube config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func (s *Service) startServiceInformer() error {
	factory := informers.NewSharedInformerFactory(s.Client, time.Second)
	stopper := make(chan struct{})
	defer close(stopper)
	// https://pkg.go.dev/k8s.io/client-go@v0.21.2/informers/core/v1#NewServiceInformer
	inf := factory.Core().V1().Services().Informer()
	inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			// "k8s.io/apimachinery/pkg/apis/meta/v1" provides an Object
			// interface that allows us to get metadata easily
			mObj := obj.(*v1.Service)
			log.Printf("service deleted: %s", mObj.GetName())
		},
		AddFunc: func(obj interface{}) {
			// "k8s.io/apimachinery/pkg/apis/meta/v1" provides an Object
			// interface that allows us to get metadata easily
			mObj := obj.(*v1.Service)
			log.Printf("New Service Added to Store: %s", mObj.GetName())
		},
	})

	inf.Run(stopper)

	return nil
}
