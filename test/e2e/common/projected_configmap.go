/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

import (
	"fmt"
	"os"
	"path"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
	imageutils "k8s.io/kubernetes/test/utils/image"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("[sig-storage] Projected configMap", func() {
	f := framework.NewDefaultFramework("projected")

	/*
	   Release : v1.9
	   Testname: Projected Volume, ConfigMap, volume mode default
	   Description: A Pod is created with projected volume source ‘ConfigMap’ to store a configMap with default permission mode. Pod MUST be able to read the content of the ConfigMap successfully and the mode on the volume MUST be -rw-r—-r—-.
	*/
	framework.ConformanceIt("should be consumable from pods in volume [NodeConformance]", func() {
		doProjectedConfigMapE2EWithoutMappings(f, 0, 0, nil)
	})

	/*
	   Release : v1.9
	   Testname: Projected Volume, ConfigMap, volume mode 0400
	   Description: A Pod is created with projected volume source ‘ConfigMap’ to store a configMap with permission mode set to 0400. Pod MUST be able to read the content of the ConfigMap successfully and the mode on the volume MUST be -r——-——-—-.
	*/
	framework.ConformanceIt("should be consumable from pods in volume with defaultMode set [NodeConformance]", func() {
		defaultMode := int32(0400)
		doProjectedConfigMapE2EWithoutMappings(f, 0, 0, &defaultMode)
	})

	It("should be consumable from pods in volume as non-root with defaultMode and fsGroup set [NodeFeature:FSGroup]", func() {
		defaultMode := int32(0440) /* setting fsGroup sets mode to at least 440 */
		doProjectedConfigMapE2EWithoutMappings(f, 1000, 1001, &defaultMode)
	})

	/*
	   Release : v1.9
	   Testname: Projected Volume, ConfigMap, non-root user
	   Description: A Pod is created with projected volume source ‘ConfigMap’ to store a configMap as non-root user with uid 1000. Pod MUST be able to read the content of the ConfigMap successfully and the mode on the volume MUST be -rw—r——r—-.
	*/
	framework.ConformanceIt("should be consumable from pods in volume as non-root [NodeConformance]", func() {
		doProjectedConfigMapE2EWithoutMappings(f, 1000, 0, nil)
	})

	It("should be consumable from pods in volume as non-root with FSGroup [NodeFeature:FSGroup]", func() {
		doProjectedConfigMapE2EWithoutMappings(f, 1000, 1001, nil)
	})

	/*
	   Release : v1.9
	   Testname: Projected Volume, ConfigMap, mapped
	   Description: A Pod is created with projected volume source ‘ConfigMap’ to store a configMap with default permission mode. The ConfigMap is also mapped to a custom path. Pod MUST be able to read the content of the ConfigMap from the custom location successfully and the mode on the volume MUST be -rw—r——r—-.
	*/
	framework.ConformanceIt("should be consumable from pods in volume with mappings [NodeConformance]", func() {
		doProjectedConfigMapE2EWithMappings(f, 0, 0, nil)
	})

	/*
	   Release : v1.9
	   Testname: Projected Volume, ConfigMap, mapped, volume mode 0400
	   Description: A Pod is created with projected volume source ‘ConfigMap’ to store a configMap with permission mode set to 0400. The ConfigMap is also mapped to a custom path. Pod MUST be able to read the content of the ConfigMap from the custom location successfully and the mode on the volume MUST be -r-—r——r—-.
	*/
	framework.ConformanceIt("should be consumable from pods in volume with mappings and Item mode set [NodeConformance]", func() {
		mode := int32(0400)
		doProjectedConfigMapE2EWithMappings(f, 0, 0, &mode)
	})

	/*
	   Release : v1.9
	   Testname: Projected Volume, ConfigMap, mapped, non-root user
	   Description: A Pod is created with projected volume source ‘ConfigMap’ to store a configMap as non-root user with uid 1000. The ConfigMap is also mapped to a custom path. Pod MUST be able to read the content of the ConfigMap from the custom location successfully and the mode on the volume MUST be -r-—r——r—-.
	*/
	framework.ConformanceIt("should be consumable from pods in volume with mappings as non-root [NodeConformance]", func() {
		doProjectedConfigMapE2EWithMappings(f, 1000, 0, nil)
	})

	It("should be consumable from pods in volume with mappings as non-root with FSGroup [NodeFeature:FSGroup]", func() {
		doProjectedConfigMapE2EWithMappings(f, 1000, 1001, nil)
	})

	/*
	   Release : v1.9
	   Testname: Projected Volume, ConfigMap, update
	   Description: A Pod is created with projected volume source ‘ConfigMap’ to store a configMap and performs a create and update to new value. Pod MUST be able to create the configMap with value-1. Pod MUST be able to update the value in the confgiMap to value-2.
	*/
	framework.ConformanceIt("updates should be reflected in volume [NodeConformance]", func() {
		podLogTimeout := framework.GetPodSecretUpdateTimeout(f.ClientSet)
		containerTimeoutArg := fmt.Sprintf("--retry_time=%v", int(podLogTimeout.Seconds()))

		name := "projected-configmap-test-upd-" + string(uuid.NewUUID())
		volumeName := "projected-configmap-volume"
		volumeMountPath := "/etc/projected-configmap-volume"
		containerName := "projected-configmap-volume-test"
		configMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: f.Namespace.Name,
				Name:      name,
			},
			Data: map[string]string{
				"data-1": "value-1",
			},
		}

		By(fmt.Sprintf("Creating projection with configMap that has name %s", configMap.Name))
		var err error
		if configMap, err = f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Create(configMap); err != nil {
			framework.Failf("unable to create test configMap %s: %v", configMap.Name, err)
		}

		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod-projected-configmaps-" + string(uuid.NewUUID()),
			},
			Spec: v1.PodSpec{
				Volumes: []v1.Volume{
					{
						Name: volumeName,
						VolumeSource: v1.VolumeSource{
							Projected: &v1.ProjectedVolumeSource{
								Sources: []v1.VolumeProjection{
									{
										ConfigMap: &v1.ConfigMapProjection{
											LocalObjectReference: v1.LocalObjectReference{
												Name: name,
											},
										},
									},
								},
							},
						},
					},
				},
				Containers: []v1.Container{
					{
						Name:    containerName,
						Image:   imageutils.GetE2EImage(imageutils.Mounttest),
						Command: []string{"/mounttest", "--break_on_expected_content=false", containerTimeoutArg, "--file_content_in_loop=/etc/projected-configmap-volume/data-1"},
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      volumeName,
								MountPath: volumeMountPath,
								ReadOnly:  true,
							},
						},
					},
				},
				RestartPolicy: v1.RestartPolicyNever,
			},
		}
		By("Creating the pod")
		f.PodClient().CreateSync(pod)

		pollLogs := func() (string, error) {
			return framework.GetPodLogs(f.ClientSet, f.Namespace.Name, pod.Name, containerName)
		}

		Eventually(pollLogs, podLogTimeout, framework.Poll).Should(ContainSubstring("value-1"))

		By(fmt.Sprintf("Updating configmap %v", configMap.Name))
		configMap.ResourceVersion = "" // to force update
		configMap.Data["data-1"] = "value-2"
		_, err = f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Update(configMap)
		Expect(err).NotTo(HaveOccurred(), "Failed to update configmap %q in namespace %q", configMap.Name, f.Namespace.Name)

		By("waiting to observe update in volume")
		Eventually(pollLogs, podLogTimeout, framework.Poll).Should(ContainSubstring("value-2"))
	})

	/*
	   Release : v1.9
	   Testname: Projected Volume, ConfigMap, create, update and delete
	   Description: Create a Pod with three containers with ConfigMaps namely a create, update and delete container. Create Container when started MUST not have configMap, update and delete containers MUST be created with a ConfigMap value as ‘value-1’. Create a configMap in the create container, the Pod MUST be able to read the configMap from the create container. Update the configMap in the update container, Pod MUST be able to read the updated configMap value. Delete the configMap in the delete container. Pod MUST fail to read the configMap from the delete container.
	*/
	framework.ConformanceIt("optional updates should be reflected in volume [NodeConformance]", func() {
		podLogTimeout := framework.GetPodSecretUpdateTimeout(f.ClientSet)
		containerTimeoutArg := fmt.Sprintf("--retry_time=%v", int(podLogTimeout.Seconds()))
		trueVal := true
		volumeMountPath := "/etc/projected-configmap-volumes"

		deleteName := "cm-test-opt-del-" + string(uuid.NewUUID())
		deleteContainerName := "delcm-volume-test"
		deleteVolumeName := "deletecm-volume"
		deleteConfigMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: f.Namespace.Name,
				Name:      deleteName,
			},
			Data: map[string]string{
				"data-1": "value-1",
			},
		}

		updateName := "cm-test-opt-upd-" + string(uuid.NewUUID())
		updateContainerName := "updcm-volume-test"
		updateVolumeName := "updatecm-volume"
		updateConfigMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: f.Namespace.Name,
				Name:      updateName,
			},
			Data: map[string]string{
				"data-1": "value-1",
			},
		}

		createName := "cm-test-opt-create-" + string(uuid.NewUUID())
		createContainerName := "createcm-volume-test"
		createVolumeName := "createcm-volume"
		createConfigMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: f.Namespace.Name,
				Name:      createName,
			},
			Data: map[string]string{
				"data-1": "value-1",
			},
		}

		By(fmt.Sprintf("Creating configMap with name %s", deleteConfigMap.Name))
		var err error
		if deleteConfigMap, err = f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Create(deleteConfigMap); err != nil {
			framework.Failf("unable to create test configMap %s: %v", deleteConfigMap.Name, err)
		}

		By(fmt.Sprintf("Creating configMap with name %s", updateConfigMap.Name))
		if updateConfigMap, err = f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Create(updateConfigMap); err != nil {
			framework.Failf("unable to create test configMap %s: %v", updateConfigMap.Name, err)
		}

		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod-projected-configmaps-" + string(uuid.NewUUID()),
			},
			Spec: v1.PodSpec{
				Volumes: []v1.Volume{
					{
						Name: deleteVolumeName,
						VolumeSource: v1.VolumeSource{
							Projected: &v1.ProjectedVolumeSource{
								Sources: []v1.VolumeProjection{
									{
										ConfigMap: &v1.ConfigMapProjection{
											LocalObjectReference: v1.LocalObjectReference{
												Name: deleteName,
											},
											Optional: &trueVal,
										},
									},
								},
							},
						},
					},
					{
						Name: updateVolumeName,
						VolumeSource: v1.VolumeSource{
							Projected: &v1.ProjectedVolumeSource{
								Sources: []v1.VolumeProjection{
									{
										ConfigMap: &v1.ConfigMapProjection{
											LocalObjectReference: v1.LocalObjectReference{
												Name: updateName,
											},
											Optional: &trueVal,
										},
									},
								},
							},
						},
					},
					{
						Name: createVolumeName,
						VolumeSource: v1.VolumeSource{
							Projected: &v1.ProjectedVolumeSource{
								Sources: []v1.VolumeProjection{
									{
										ConfigMap: &v1.ConfigMapProjection{
											LocalObjectReference: v1.LocalObjectReference{
												Name: createName,
											},
											Optional: &trueVal,
										},
									},
								},
							},
						},
					},
				},
				Containers: []v1.Container{
					{
						Name:    deleteContainerName,
						Image:   imageutils.GetE2EImage(imageutils.Mounttest),
						Command: []string{"/mounttest", "--break_on_expected_content=false", containerTimeoutArg, "--file_content_in_loop=/etc/projected-configmap-volumes/delete/data-1"},
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      deleteVolumeName,
								MountPath: path.Join(volumeMountPath, "delete"),
								ReadOnly:  true,
							},
						},
					},
					{
						Name:    updateContainerName,
						Image:   imageutils.GetE2EImage(imageutils.Mounttest),
						Command: []string{"/mounttest", "--break_on_expected_content=false", containerTimeoutArg, "--file_content_in_loop=/etc/projected-configmap-volumes/update/data-3"},
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      updateVolumeName,
								MountPath: path.Join(volumeMountPath, "update"),
								ReadOnly:  true,
							},
						},
					},
					{
						Name:    createContainerName,
						Image:   imageutils.GetE2EImage(imageutils.Mounttest),
						Command: []string{"/mounttest", "--break_on_expected_content=false", containerTimeoutArg, "--file_content_in_loop=/etc/projected-configmap-volumes/create/data-1"},
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      createVolumeName,
								MountPath: path.Join(volumeMountPath, "create"),
								ReadOnly:  true,
							},
						},
					},
				},
				RestartPolicy: v1.RestartPolicyNever,
			},
		}
		By("Creating the pod")
		f.PodClient().CreateSync(pod)

		pollCreateLogs := func() (string, error) {
			return framework.GetPodLogs(f.ClientSet, f.Namespace.Name, pod.Name, createContainerName)
		}
		Eventually(pollCreateLogs, podLogTimeout, framework.Poll).Should(ContainSubstring("Error reading file /etc/projected-configmap-volumes/create/data-1"))

		pollUpdateLogs := func() (string, error) {
			return framework.GetPodLogs(f.ClientSet, f.Namespace.Name, pod.Name, updateContainerName)
		}
		Eventually(pollUpdateLogs, podLogTimeout, framework.Poll).Should(ContainSubstring("Error reading file /etc/projected-configmap-volumes/update/data-3"))

		pollDeleteLogs := func() (string, error) {
			return framework.GetPodLogs(f.ClientSet, f.Namespace.Name, pod.Name, deleteContainerName)
		}
		Eventually(pollDeleteLogs, podLogTimeout, framework.Poll).Should(ContainSubstring("value-1"))

		By(fmt.Sprintf("Deleting configmap %v", deleteConfigMap.Name))
		err = f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Delete(deleteConfigMap.Name, &metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to delete configmap %q in namespace %q", deleteConfigMap.Name, f.Namespace.Name)

		By(fmt.Sprintf("Updating configmap %v", updateConfigMap.Name))
		updateConfigMap.ResourceVersion = "" // to force update
		delete(updateConfigMap.Data, "data-1")
		updateConfigMap.Data["data-3"] = "value-3"
		_, err = f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Update(updateConfigMap)
		Expect(err).NotTo(HaveOccurred(), "Failed to update configmap %q in namespace %q", updateConfigMap.Name, f.Namespace.Name)

		By(fmt.Sprintf("Creating configMap with name %s", createConfigMap.Name))
		if createConfigMap, err = f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Create(createConfigMap); err != nil {
			framework.Failf("unable to create test configMap %s: %v", createConfigMap.Name, err)
		}

		By("waiting to observe update in volume")

		Eventually(pollCreateLogs, podLogTimeout, framework.Poll).Should(ContainSubstring("value-1"))
		Eventually(pollUpdateLogs, podLogTimeout, framework.Poll).Should(ContainSubstring("value-3"))
		Eventually(pollDeleteLogs, podLogTimeout, framework.Poll).Should(ContainSubstring("Error reading file /etc/projected-configmap-volumes/delete/data-1"))
	})

	/*
	   Release : v1.9
	   Testname: Projected Volume, ConfigMap, multiple volume paths
	   Description: A Pod is created with a projected volume source ‘ConfigMap’ to store a configMap. The configMap is mapped to two different volume mounts. Pod MUST be able to read the content of the configMap successfully from the two volume mounts.
	*/
	framework.ConformanceIt("should be consumable in multiple volumes in the same pod [NodeConformance]", func() {
		var (
			name             = "projected-configmap-test-volume-" + string(uuid.NewUUID())
			volumeName       = "projected-configmap-volume"
			volumeMountPath  = "/etc/projected-configmap-volume"
			volumeName2      = "projected-configmap-volume-2"
			volumeMountPath2 = "/etc/projected-configmap-volume-2"
			configMap        = newConfigMap(f, name)
		)

		By(fmt.Sprintf("Creating configMap with name %s", configMap.Name))
		var err error
		if configMap, err = f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Create(configMap); err != nil {
			framework.Failf("unable to create test configMap %s: %v", configMap.Name, err)
		}

		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod-projected-configmaps-" + string(uuid.NewUUID()),
			},
			Spec: v1.PodSpec{
				Volumes: []v1.Volume{
					{
						Name: volumeName,
						VolumeSource: v1.VolumeSource{
							Projected: &v1.ProjectedVolumeSource{
								Sources: []v1.VolumeProjection{
									{
										ConfigMap: &v1.ConfigMapProjection{
											LocalObjectReference: v1.LocalObjectReference{
												Name: name,
											},
										},
									},
								},
							},
						},
					},
					{
						Name: volumeName2,
						VolumeSource: v1.VolumeSource{
							Projected: &v1.ProjectedVolumeSource{
								Sources: []v1.VolumeProjection{
									{

										ConfigMap: &v1.ConfigMapProjection{
											LocalObjectReference: v1.LocalObjectReference{
												Name: name,
											},
										},
									},
								},
							},
						},
					},
				},
				Containers: []v1.Container{
					{
						Name:  "projected-configmap-volume-test",
						Image: imageutils.GetE2EImage(imageutils.Mounttest),
						Args:  []string{"--file_content=/etc/projected-configmap-volume/data-1"},
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      volumeName,
								MountPath: volumeMountPath,
								ReadOnly:  true,
							},
							{
								Name:      volumeName2,
								MountPath: volumeMountPath2,
								ReadOnly:  true,
							},
						},
					},
				},
				RestartPolicy: v1.RestartPolicyNever,
			},
		}

		f.TestContainerOutput("consume configMaps", pod, 0, []string{
			"content of file \"/etc/projected-configmap-volume/data-1\": value-1",
		})

	})
})

func doProjectedConfigMapE2EWithoutMappings(f *framework.Framework, uid, fsGroup int64, defaultMode *int32) {
	userID := int64(uid)
	groupID := int64(fsGroup)

	var (
		name            = "projected-configmap-test-volume-" + string(uuid.NewUUID())
		volumeName      = "projected-configmap-volume"
		volumeMountPath = "/etc/projected-configmap-volume"
		configMap       = newConfigMap(f, name)
	)

	By(fmt.Sprintf("Creating configMap with name %s", configMap.Name))
	var err error
	if configMap, err = f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Create(configMap); err != nil {
		framework.Failf("unable to create test configMap %s: %v", configMap.Name, err)
	}

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-projected-configmaps-" + string(uuid.NewUUID()),
		},
		Spec: v1.PodSpec{
			SecurityContext: &v1.PodSecurityContext{},
			Volumes: []v1.Volume{
				{
					Name: volumeName,
					VolumeSource: v1.VolumeSource{
						Projected: &v1.ProjectedVolumeSource{
							Sources: []v1.VolumeProjection{
								{
									ConfigMap: &v1.ConfigMapProjection{
										LocalObjectReference: v1.LocalObjectReference{
											Name: name,
										},
									},
								},
							},
						},
					},
				},
			},
			Containers: []v1.Container{
				{
					Name:  "projected-configmap-volume-test",
					Image: imageutils.GetE2EImage(imageutils.Mounttest),
					Args: []string{
						"--file_content=/etc/projected-configmap-volume/data-1",
						"--file_mode=/etc/projected-configmap-volume/data-1"},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      volumeName,
							MountPath: volumeMountPath,
						},
					},
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
		},
	}

	if userID != 0 {
		pod.Spec.SecurityContext.RunAsUser = &userID
	}

	if groupID != 0 {
		pod.Spec.SecurityContext.FSGroup = &groupID
	}

	if defaultMode != nil {
		//pod.Spec.Volumes[0].VolumeSource.Projected.Sources[0].ConfigMap.DefaultMode = defaultMode
		pod.Spec.Volumes[0].VolumeSource.Projected.DefaultMode = defaultMode
	} else {
		mode := int32(0644)
		defaultMode = &mode
	}

	modeString := fmt.Sprintf("%v", os.FileMode(*defaultMode))
	output := []string{
		"content of file \"/etc/projected-configmap-volume/data-1\": value-1",
		"mode of file \"/etc/projected-configmap-volume/data-1\": " + modeString,
	}
	f.TestContainerOutput("consume configMaps", pod, 0, output)
}

func doProjectedConfigMapE2EWithMappings(f *framework.Framework, uid, fsGroup int64, itemMode *int32) {
	userID := int64(uid)
	groupID := int64(fsGroup)

	var (
		name            = "projected-configmap-test-volume-map-" + string(uuid.NewUUID())
		volumeName      = "projected-configmap-volume"
		volumeMountPath = "/etc/projected-configmap-volume"
		configMap       = newConfigMap(f, name)
	)

	By(fmt.Sprintf("Creating configMap with name %s", configMap.Name))

	var err error
	if configMap, err = f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Create(configMap); err != nil {
		framework.Failf("unable to create test configMap %s: %v", configMap.Name, err)
	}

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-projected-configmaps-" + string(uuid.NewUUID()),
		},
		Spec: v1.PodSpec{
			SecurityContext: &v1.PodSecurityContext{},
			Volumes: []v1.Volume{
				{
					Name: volumeName,
					VolumeSource: v1.VolumeSource{
						Projected: &v1.ProjectedVolumeSource{
							Sources: []v1.VolumeProjection{
								{
									ConfigMap: &v1.ConfigMapProjection{
										LocalObjectReference: v1.LocalObjectReference{
											Name: name,
										},
										Items: []v1.KeyToPath{
											{
												Key:  "data-2",
												Path: "path/to/data-2",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			Containers: []v1.Container{
				{
					Name:  "projected-configmap-volume-test",
					Image: imageutils.GetE2EImage(imageutils.Mounttest),
					Args: []string{"--file_content=/etc/projected-configmap-volume/path/to/data-2",
						"--file_mode=/etc/projected-configmap-volume/path/to/data-2"},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      volumeName,
							MountPath: volumeMountPath,
							ReadOnly:  true,
						},
					},
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
		},
	}

	if userID != 0 {
		pod.Spec.SecurityContext.RunAsUser = &userID
	}

	if groupID != 0 {
		pod.Spec.SecurityContext.FSGroup = &groupID
	}

	if itemMode != nil {
		//pod.Spec.Volumes[0].VolumeSource.ConfigMap.Items[0].Mode = itemMode
		pod.Spec.Volumes[0].VolumeSource.Projected.DefaultMode = itemMode
	} else {
		mode := int32(0644)
		itemMode = &mode
	}

	// Just check file mode if fsGroup is not set. If fsGroup is set, the
	// final mode is adjusted and we are not testing that case.
	output := []string{
		"content of file \"/etc/projected-configmap-volume/path/to/data-2\": value-2",
	}
	if fsGroup == 0 {
		modeString := fmt.Sprintf("%v", os.FileMode(*itemMode))
		output = append(output, "mode of file \"/etc/projected-configmap-volume/path/to/data-2\": "+modeString)
	}
	f.TestContainerOutput("consume configMaps", pod, 0, output)
}
