# Kubernetes-Controller
Repository for a Kubernetes Custom Controller.
In this sample, I have made a controller which checks for Pods in all namespaces and sees whether a pod has been created, updated or deleted and prints the name of the pod. This can be used to check for any other resources as well 

### How to Run

  - First install glide, and in the directory run `glide update` to fetch all the dependencies
  - Then run `go build`
  - And then run `./Kubernetes-Controller`