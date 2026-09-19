# N6 Reboot Gate

Status: NOT RUN

The only supplied server contains the existing N5 installation and database.
A real OS reboot was intentionally not performed because it would disrupt an
active non-isolated installation and would not constitute an N6 clean-server
reboot test. The isolated mount namespace used for fresh install cannot
survive an OS reboot.

Required follow-up: provide a disposable clean Debian host, then run the N6
installer, verify service enablement and SS2022 TCP/UDP after `/sbin/reboot`,
and record the result as `n6-local-07-reboot-uat`.
