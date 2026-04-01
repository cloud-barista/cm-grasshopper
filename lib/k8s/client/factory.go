package client

import (
	"context"
	"fmt"
	"log"
	"strings"

	k8scommon "github.com/cloud-barista/cm-grasshopper/lib/k8s/common"
	restmodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewRESTConfig(cluster *restmodel.ClusterAccess) (*rest.Config, error) {
	if err := k8scommon.ValidateClusterAccess(cluster); err != nil {
		return nil, err
	}

	kubeconfig, err := k8scommon.DecodeKubeconfig(cluster.Kubeconfig)
	if err != nil {
		return nil, err
	}

	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	cfg.QPS = 50
	cfg.Burst = 100
	return cfg, nil
}

func NewScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := velerov1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	return scheme, nil
}

func NewKubernetesClient(cluster *restmodel.ClusterAccess) (*kubernetes.Clientset, ctrlclient.Client, error) {
	cfg, err := NewRESTConfig(cluster)
	if err != nil {
		return nil, nil, err
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	scheme, err := NewScheme()
	if err != nil {
		return nil, nil, err
	}

	controllerClient, err := ctrlclient.New(cfg, ctrlclient.Options{Scheme: scheme})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create controller client: %w", err)
	}

	return clientset, controllerClient, nil
}

func NewHelmActionConfig(cluster *restmodel.ClusterAccess) (*action.Configuration, error) {
	cfg, err := NewRESTConfig(cluster)
	if err != nil {
		return nil, err
	}

	namespace := k8scommon.DefaultNamespace(cluster, "default")
	flags := genericclioptions.NewConfigFlags(false)
	flags.Namespace = &namespace
	flags.WrapConfigFn = func(_ *rest.Config) *rest.Config {
		return cfg
	}

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(flags, namespace, "secret", log.Printf); err != nil {
		return nil, fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	return actionConfig, nil
}

func NewMinIOClient(minioAccess *restmodel.MinIOAccess) (*minio.Client, error) {
	if err := k8scommon.ValidateMinIOAccess(minioAccess); err != nil {
		return nil, err
	}

	endpoint, useSSL, err := k8scommon.NormalizeMinIOEndpoint(minioAccess)
	if err != nil {
		return nil, err
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(minioAccess.AccessKey, minioAccess.SecretKey, ""),
		Secure:       useSSL,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return client, nil
}

func EnsureMinIOBucket(ctx context.Context, minioClient *minio.Client, bucketName string) error {
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		errorResponse := minio.ToErrorResponse(err)
		switch errorResponse.Code {
		case "NoSuchBucket":
			exists = false
		case "NotFound":
			exists = false
		default:
			if strings.Contains(strings.ToLower(err.Error()), "bucket does not exist") {
				exists = false
				break
			}
			return err
		}
	}
	if exists {
		return nil
	}
	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		errorResponse := minio.ToErrorResponse(err)
		switch errorResponse.Code {
		case "BucketAlreadyOwnedByYou":
			return nil
		case "BucketAlreadyExists":
			return nil
		default:
			return err
		}
	}

	return nil
}
