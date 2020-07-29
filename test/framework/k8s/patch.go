package k8s

import (
	"encoding/json"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

func CreateStrategicTwoWayMergePatch(oldObj metav1.Object, newObj metav1.Object, objType interface{}) ([]byte, error) {
	oldData, err := json.Marshal(oldObj)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to Marshal oldObj: %s/%s", oldObj.GetName(), oldObj.GetName())
	}

	newData, err := json.Marshal(newObj)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to Marshal newObj: %s/%s", newObj.GetName(), newObj.GetName())
	}

	return strategicpatch.CreateTwoWayMergePatch(oldData, newData, objType)
}

func CreateJSONMergePatch(oldObj metav1.Object, newObj metav1.Object, _ interface{}) ([]byte, error) {
	oldData, err := json.Marshal(oldObj)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to Marshal oldObj: %s/%s", oldObj.GetName(), oldObj.GetName())
	}

	newData, err := json.Marshal(newObj)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to Marshal newObj: %s/%s", newObj.GetName(), newObj.GetName())
	}

	return jsonpatch.CreateMergePatch(oldData, newData)
}
