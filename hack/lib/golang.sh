#!/bin/bash

# This utility file contains go related initialization details.

# Need to enable the vendor experiment so tools such as codecgen can function
# when generating references to vendored packages, such as k8s.io/kubernetes/...
export GO15VENDOREXPERIMENT=1
