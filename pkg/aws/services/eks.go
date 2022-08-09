package services

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
)

type EKS interface {
	eksiface.EKSAPI
}

// NewAppMesh constructs new AppMesh implementation.
func NewEKS(session *session.Session) EKS {
	return &defaultEKS{
		EKSAPI: eks.New(session),
	}
}

type defaultEKS struct {
	eksiface.EKSAPI
}
