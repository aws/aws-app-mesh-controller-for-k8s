package conversions

import (
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
)

func Convert_CRD_Labls_To_SDK_TagChangeSet(crdLabels map[string]string, sdkTags []*appmeshsdk.TagRef) (map[string][]*appmeshsdk.TagRef, error) {
	add := make(map[string]string)
	remove := make(map[string]string)
	existing := make(map[string]string)

	for _, tag := range sdkTags {
		k := *tag.Key
		v := *tag.Value
		existing[k] = v

		if _, found := crdLabels[k]; !found {
			remove[k] = v
		}
	}

	for k, v := range crdLabels {
		if current, found := existing[k]; found {
			if v != current {
				add[k] = v
			}
		} else {
			add[k] = v
		}
	}

	sdkAdd, err := Convert_CRD_Labels_To_SDK_Tags(add)
	if err != nil {
		return nil, err
	}

	sdkRemove, err := Convert_CRD_Labels_To_SDK_Tags(remove)
	if err != nil {
		return nil, err
	}

	sdkChangeset := make(map[string][]*appmeshsdk.TagRef)
	sdkChangeset["add"] = sdkAdd
	sdkChangeset["remove"] = sdkRemove

	return sdkChangeset, nil
}

func Convert_CRD_Labels_To_SDK_Tags(crdObj map[string]string) ([]*appmeshsdk.TagRef, error) {
	sdkObj := []*appmeshsdk.TagRef{}
	for k, v := range crdObj {
		sdkObj = append(sdkObj, &appmeshsdk.TagRef{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return sdkObj, nil
}
