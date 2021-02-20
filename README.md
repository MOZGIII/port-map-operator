# port-map-operator

A `LoadBalancer` `Service` type implementation for small home clusters.

Maps the ports from your router to a Kubernetes cluster node
via the [Port Control Protocol](https://tools.ietf.org/html/rfc6887).

It does not perform real load balancing of any kind, but just takes care of
the port forwarding so traffic can reach the cluster node.
Kubernetes still does its internal service-level load balancing.
