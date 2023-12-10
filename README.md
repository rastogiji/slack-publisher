# Slack Publisher
This is Kubernetes Controller which keeps a watch on the *Deployment* objects on the cluster and if any deployment has a container that doesn't have resource requests and limits, it sends a Slack alert.

Environment Variables needed:
- TOKEN
- CHANNEL

