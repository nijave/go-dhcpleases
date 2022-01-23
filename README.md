# go-dhcpleases

Watches isc-dhcp lease file for changes, parses hostnames and their IPs, updates a hosts file, and notifies dnsmasq.

See https://github.com/opnsense/ports/blob/master/opnsense/dhcpleases/files/dhcpleases.c

## Differences
 - Only use last lease information for a given IP in the file (isc-dhcp appends lease info "transaction" style so a DHCPRELEASE appears as a new entry at the end of the file while the original entry remains in-tact. Once an hour, isc-dhcp reconciles this file. In the meantime, there can be multiple entries for the same lease)
 - Only use the most recent hostname if a hostname has multiple leases (useful for devices with changing NICs like a laptop with wired/wireless or a VM getting deleted and recreated with a new NIC)

## Todo
 - Shutdown signal handling
 - Tests
 - Handle filenames as flags (like dhcpleases.c)
