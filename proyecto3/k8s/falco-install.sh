#!/bin/bash
helm repo add falcosecurity https://falcosecurity.github.io/charts
helm repo update
helm upgrade --install falco falcosecurity/falco -n falco --create-namespace \
  --set driver.kind=modern_ebpf --set tty=true --set falcosidekick.enabled=false \
  --set falco.rule_matching=all \
  -f k8s/falco-rules.yaml
