package main

import (
	//"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strings"

	"github.com/asiainfoLDP/datafoundry_proxy/messages"
	"github.com/asiainfoLDP/datafoundry_proxy/openshift"
	"github.com/asiainfoLDP/datahub_commons/common"
	kapiresource "k8s.io/kubernetes/pkg/api/resource"
	kapi "k8s.io/kubernetes/pkg/api/v1"

	heketi "github.com/heketi/heketi/client/api/go-client"
	"github.com/heketi/heketi/pkg/glusterfs/api"
)

const (
	MinVolumnSize = 10
	MaxVolumnSize = 200

	Gi = 2 << 30
)

var invalidVolumnSize = fmt.Errorf(
	"volumn size must in range [%d, %d]",
	MinVolumnSize, MaxVolumnSize)

func heketiClient() *heketi.Client {
	return heketi.NewClient(
		fmt.Sprintf("http://%s:%s",
			HeketiEnv.Get(HEKETI_HOST_ADDR),
			HeketiEnv.Get(HEKETI_HOST_PORT),
		),
		HeketiEnv.Get(HEKETI_USER),
		HeketiEnv.Get(HEKETI_KEY),
	)
}

func glusterEndpointsName() string {
	return GlusterEnv.Get(GLUSTER_ENDPOINTS_NAME)
}

//==============================================================
//
//==============================================================

func PvcName2PvName(namespace, volName string) string {
	return fmt.Sprintf("%s-%s", namespace, volName) // don't change
}

func VolumeId2VolumeName(volId string) string {
	return "vol_" + volId
}

func VolumeName2VolumeId(volName string) string {
	const prefix = "vol_"
	if strings.HasPrefix(volName, prefix) {
		return volName[len(prefix):]
	}

	return volName
}

//func PvName2VolumeName(namespace, pvName string) string {
//	prefix := PvcName2PvName(pvName, "")
//	if strings.HasPrefix(pvName, prefix) {
//		return pvName[len(prefix):]
//	}
//
//	return pvName
//}

//==============================================================
//
//==============================================================

//==============================================================
//
//==============================================================

func CreateVolume(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	glog.Infoln("from", r.RemoteAddr, r.Method, r.URL.RequestURI(), r.Proto)

	var username string
	var err error

	if username, err = authedIdentities(r); err != nil {
		RespError(w, err, http.StatusUnauthorized)
		return
	}

	// params

	namespace, e := messages.MustStringParamInPath(params, "namespace", messages.StringParamType_UrlWord)
	if e != nil {
		RespError(w, e, http.StatusBadRequest)
		return
	}

	m, err := common.ParseRequestJsonAsMap(r)
	if err != nil {
		glog.Error(err)
		RespError(w, err, http.StatusBadRequest)
		return
	}

	size, e := messages.MustIntParamInMap(m, "size")
	if e != nil {
		RespError(w, e, http.StatusBadRequest)
		return
	}
	if size < MinVolumnSize || size > MaxVolumnSize {
		RespError(w, invalidVolumnSize, http.StatusBadRequest)
		return
	}

	pvcname, e := messages.MustStringParamInMap(m, "name", messages.StringParamType_UrlWord)
	if e != nil {
		RespError(w, e, http.StatusBadRequest)
		return
	}
	valid, msg := NameIsDNSLabel(pvcname, false)
	if !valid {
		RespError(w, errors.New(msg), http.StatusBadRequest)
		return
	}

	// todo: check permission

	_ = username
	_ = namespace

	// create volumn

	hkiClient := heketiClient()

	// clusterlist, err := hkiClient.ClusterList()
	// if err != nil {
	// 	glog.Error(err)
	// 	RespError(w, err, http.StatusBadRequest)
	// 	return
	// }

	req := &api.VolumeCreateRequest{}
	req.Size = int(size)
	//req.Name = pvcname // ! don't set name, otherwise, can't get volume id from pv

	req.Clusters = []string{"68aa170df797272ac2ac90fac1f7460b"} //hacked by san
	req.Durability.Type = api.DurabilityReplicate
	req.Durability.Replicate.Replica = 3
	req.Durability.Disperse.Data = 4
	req.Durability.Disperse.Redundancy = 2

	// if snapshotFactor > 1.0 {
	//	req.Snapshot.Factor = float32(snapshotFactor)
	//	req.Snapshot.Enable = true
	// }

	var succeeded = false

	volume, err := hkiClient.VolumeCreate(req)
	if err != nil {
		glog.Error(err)
		RespError(w, err, http.StatusBadRequest)
		return
	}

	defer func() {
		if succeeded {
			return
		}

		err := hkiClient.VolumeDelete(volume.Id)
		if err != nil {
			glog.Warningf("delete volume (%s, %s) on failed to CreateVolume", pvcname, volume.Id)
		}
	}()

	// create pv

	openshiftUrlPrefix := "" //pv is a cluster scoped resouece. "/namespaces/" + namespace

	resourceList := make(kapi.ResourceList)
	resourceList[kapi.ResourceStorage] = *kapiresource.NewQuantity(int64(size*Gi), kapiresource.BinarySI)

	inputPV := &kapi.PersistentVolume{}
	{
		inputPV.Kind = "PersistentVolume"
		inputPV.APIVersion = "v1"
		inputPV.Name = PvcName2PvName(namespace, pvcname)
		inputPV.Spec.Capacity = resourceList
		inputPV.Spec.PersistentVolumeSource = kapi.PersistentVolumeSource{
			Glusterfs: &kapi.GlusterfsVolumeSource{
				EndpointsName: glusterEndpointsName(),
				Path:          VolumeId2VolumeName(volume.Id),
			},
		}
		inputPV.Spec.AccessModes = []kapi.PersistentVolumeAccessMode{
			kapi.ReadWriteMany,
		}
		inputPV.Spec.PersistentVolumeReclaimPolicy = kapi.PersistentVolumeReclaimRecycle
	}

	outputPV := &kapi.PersistentVolume{}
	osrPV := openshift.NewOpenshiftREST(nil)
	osrPV.KPost(openshiftUrlPrefix+"/persistentvolumes", inputPV, outputPV)
	if osrPV.Err != nil {
		glog.Warningf("create pv error CreateVolume: pvname=%s, error: %s", inputPV.Name, osrPV.Err)

		RespError(w, osrPV.Err, http.StatusBadRequest)
		return
	}
	defer func() {
		if succeeded {
			return
		}

		osrPV := openshift.NewOpenshiftREST(nil)
		osrPV.KDelete(openshiftUrlPrefix+"/persistentvolumes", inputPV)
		if osrPV.Err != nil {
			glog.Warningf("delete pv error on failed to CreateVolume: pvname=%s, error: %s", inputPV.Name, osrPV.Err)
		}
	}()

	// create pvc

	inputPVC := &kapi.PersistentVolumeClaim{}
	{
		inputPV.Kind = "PersistentVolumeClaim"
		inputPV.APIVersion = "v1"
		inputPVC.Name = pvcname
		inputPVC.Spec.AccessModes = []kapi.PersistentVolumeAccessMode{
			kapi.ReadWriteMany,
		}
		inputPVC.Spec.Resources = kapi.ResourceRequirements{
			Requests: resourceList,
		}
	}

	outputPVC := &kapi.PersistentVolumeClaim{}
	osrPVC := openshift.NewOpenshiftREST(openshift.NewOpenshiftClient(retrieveToken(r)))
	osrPVC.KPost(openshiftUrlPrefix+"/persistentvolumeclaims", &inputPVC, &outputPVC)
	if osrPVC.Err != nil {
		glog.Warningf("create pvc error on CreateVolume: pvcname=%s, error: %s", pvcname, osrPVC.Err)

		RespError(w, osrPVC.Err, http.StatusBadRequest)
		return
	}
	defer func() {
		if succeeded {
			return
		}

		osrPVC := openshift.NewOpenshiftREST(openshift.NewOpenshiftClient(retrieveToken(r)))
		osrPVC.KDelete(openshiftUrlPrefix+"/persistentvolumeclaims", inputPVC)
		if osrPVC.Err != nil {
			glog.Warningf("delete pvc error on failed to CreateVolume: pvcname=%s, error: %s", pvcname, osrPVC.Err)
		}
	}()

	// ...

	succeeded = true

	RespAccepted(w, nil)
}

func DeleteVolume(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	glog.Infoln("from", r.RemoteAddr, r.Method, r.URL.RequestURI(), r.Proto)

	var username string
	var err error

	if username, err = authedIdentities(r); err != nil {
		RespError(w, err, http.StatusUnauthorized)
		return
	}

	// ...

	namespace, e := messages.MustStringParamInPath(params, "namespace", messages.StringParamType_UrlWord)
	if e != nil {
		RespError(w, e, http.StatusBadRequest)
		return
	}

	pvcname, e := messages.MustStringParamInPath(params, "name", messages.StringParamType_UrlWord)
	if e != nil {
		RespError(w, e, http.StatusBadRequest)
		return
	}
	valid, msg := NameIsDNSLabel(pvcname, false)
	if !valid {
		RespError(w, errors.New(msg), http.StatusBadRequest)
		return
	}

	// todo: check permission

	_ = username
	_ = namespace

	// get pv (will delete it at the end, for it stores the volumn id info)

	openshiftUrlPrefix := "/namespaces/" + namespace

	pvName := PvcName2PvName(namespace, pvcname)
	pv := &kapi.PersistentVolume{}
	osrGetPV := openshift.NewOpenshiftREST(nil)
	osrGetPV.KGet(openshiftUrlPrefix+"/persistentvolumes/"+pvName, pv)
	if osrGetPV.Err != nil {
		RespError(w, osrGetPV.Err, http.StatusBadRequest)
		return
	}

	// delete pvc

	go func() {
		osrDeletePVC := openshift.NewOpenshiftREST(openshift.NewOpenshiftClient(retrieveToken(r)))
		osrDeletePVC.KDelete(openshiftUrlPrefix+"/persistentvolumeclaims/"+pvcname, nil)
		if osrDeletePVC.Err != nil {
			// todo: retry

			glog.Warningf("delete pvc error: pvcname=%s, error: %s", pvcname, osrDeletePVC.Err)
		}
	}()

	// delete volume

	go func() {
		hkiClient := heketiClient()

		glusterfs := pv.Spec.PersistentVolumeSource.Glusterfs
		if glusterfs != nil {
			volId := VolumeName2VolumeId(glusterfs.Path)
			err := hkiClient.VolumeDelete(volId)
			if err != nil {
				// todo: retry

				glog.Warningf("delete volume error: pvcname=%s, volid=%s, error: %s", pvcname, volId, err)
			}
		} else {
			glog.Warningf("pv.Spec.PersistentVolumeSource.Glusterfs == nil. pvcname=%s", pvcname)
		}
	}()

	// delete pv

	osrDeletePV := openshift.NewOpenshiftREST(nil)
	osrDeletePV.KDelete(openshiftUrlPrefix+"/persistentvolumes/"+pvName, nil)
	if osrGetPV.Err != nil {
		// todo: retry once?

		glog.Warningf("delete pvc error: pvcname=%s, error: %s", pvName, osrGetPV.Err)

		RespError(w, osrDeletePV.Err, http.StatusBadRequest)
		return
	}

	// ...

	RespOK(w, nil)
}

//===============================================================
