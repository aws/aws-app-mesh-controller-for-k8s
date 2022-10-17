package services

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/aws/aws-sdk-go/service/appmesh/appmeshiface"
	ctrl "sigs.k8s.io/controller-runtime"
)

var setupLog = ctrl.Log.WithName("setup")

type AppMesh interface {
	appmeshiface.AppMeshAPI
}

// NewAppMesh constructs new AppMesh implementation.
func NewAppMesh(session *session.Session) AppMesh {
	var appMeshSession = appmesh.New(session)
	setupLog.Info("Endpoint used for AppMesh", "Endpoint", appMeshSession.Endpoint)
	return &defaultAppMesh{
		AppMeshAPI: appMeshSession,
	}
}

type defaultAppMesh struct {
	appmeshiface.AppMeshAPI
}
