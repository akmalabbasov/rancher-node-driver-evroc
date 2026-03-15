package evrocdriver

import "fmt"

func cloudInit(name string) string {
	return fmt.Sprintf(`#cloud-config
package_update: true
package_upgrade: true

hostname: %s
manage_etc_hosts: true

packages:
  - ca-certificates
  - curl
  - jq
  - qemu-guest-agent

write_files:
  - path: /etc/modules-load.d/k8s.conf
    permissions: '0644'
    content: |
      overlay
      br_netfilter
  - path: /etc/sysctl.d/99-kubernetes-cri.conf
    permissions: '0644'
    content: |
      net.bridge.bridge-nf-call-iptables = 1
      net.bridge.bridge-nf-call-ip6tables = 1
      net.ipv4.ip_forward = 1

runcmd:
  - systemctl enable --now qemu-guest-agent
  - modprobe overlay
  - modprobe br_netfilter
  - sysctl --system
`, name)
}
