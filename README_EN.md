# N5-UI

[简体中文](./README.md)| ENGLISH  

## N5-UI v0.2.0 Stable

This release was validated on a fresh Debian 11 Bullseye amd64/x86_64 system, including installation, real reboot, desktop and mobile browser checks, TCP/UDP, subscriptions, Access IP, and database consistency.

The official runtime is the bundled N5-UI Xray 26.5.3 amd64 binary.
SHA256: `128f9c34811ee74b3770eef7010d011e3946e85dfab28f2ed1804e380461b05e`

N5-UI is an independent fork based on n3-ui, and its install/release source is now fully served from `torr9522/n5-ui`.  
Runtime compatibility with the `x-ui` ecosystem is intentionally preserved: service name, command name, paths, database, API, and Xray invocation logic remain unchanged.  
N5-UI is a webUI panel based on Xray-core which supports multi protocols and multi users  
This project is a fork of [vaxilu&#39;s project](https://github.com/vaxilu/x-ui),and it is a experiental project which used by myself for learning golang   
If you need more language options ,please open a issue and let me know that

## Branding Notes

- Project brand: `n5-ui`
- Runtime identity: `x-ui` compatible
- Install command, binary name, service name, API path, and database layout are unchanged in this phase

## Developer / Maintainer Documentation

- [Codex handoff](./docs/CODEX_HANDOFF.md)
- [Development skill tree](./docs/DEVELOPMENT_SKILL_TREE.md)
- [Public development history](./docs/history/README.md)

# Changes   
- 2026.08.08：Create the independent `n5-ui` project and finish phase-1 branding migration while keeping `x-ui` runtime compatibility intact  
- 2023.07.18：Random Reality dest and serverNames;more detailed sniffing settings available  
- 2023.06.10：Enable TLS will reuse panel's certs and domain;add setting for ocspStapling;refactor device limit  
- 2023.04.09：Support REALITY for now  
- 2023.03.05：User expiry time limit for each user  
- 2023.02.09：User traffic limit for each user,support utls sharing link  
- 2022.12.07：Add device limit and more tls configuration  
- 2022.11.15：Add xtls-rprx-vision flow option;cron job for geo update and log clear    
- 2022.10.23：Fully support for English,add export links,add CPU cores display
- 2022.08.11：Support multi users on the same port;add CPU limit exceed  alert  
- 2022.07.28：Add acme standalone mode for cert issue；add  mechanism to keep X-UI alive even there exist crashes
- 2022.07.24：Add base path auto generate feature for security;add traffice reset automatically;add device alert
- 2022.07.21：Add more translations;add restart/stop xray service in Web panel
- 2022.07.11：Add time expiration notify for each inbound;add traffic limit notify for each inbound;add get url link command/inbound copy command in telegram bot  
- 2022.07.03：Add transport options in Trojan protocol;restruct Telegram bot for convenience  
- 2022.06.19：Add shadowsocks 2022 Ciphers,add inbounds search,traffic clear function in WebUI
- 2022.05.14：Add Telegram bot commands,support enable/disable/delete/status check
- 2022.04.25：Add SSH login notify
- 2022.04.23：Add WebUi login notify
- 2022.04.16：Add Telegram bot set up in WebUi pannel
- 2022.04.12：Optimize Telegram bot notify,more human friendly
- 2022.04.06：Add cert issue function，optimize installation/update and add telegram bot notify

# Basics

- support system status info check
- support multi protocols and multi users
- support protocols：vmess、vless、trojan、shadowsocks、dokodemo-door、socks、http
- support many transport method including tcp、udp、ws、kcp etc
- traffic counting,traffic restrict and time restrcit
- support custom configuration template
- support https access fot WebUI
- support SSL cert issue by Acme
- support telegram bot notify and control
- N5 egress, egress rules, traffic rules, and custom rule groups
- ALL, AI, Game, and Streaming routing
- deterministic routing priority and conflict warnings
- Subscription Lite and Access IP
- more functions in control menu  

for more detailed usages,plz see [WIKI](https://github.com/torr9522/n5-ui/wiki)

# Installation
Make sure your system `bash`, `curl`, and network are ready. The validated environment is Debian 11 Bullseye amd64/x86_64. Source installs will auto-install the Go toolchain when needed. The current embedded version is `Go 1.22.7`, with a minimum requirement of `Go 1.16+`.

Clean Install is recommended. In-place upgrade compatibility is not claimed or guaranteed for this release. Before changing versions, back up `/etc/x-ui`, the database, and important configuration.

```
bash <(curl -Ls https://raw.githubusercontent.com/torr9522/n5-ui/v0.2.0/install.sh)
```  
For English Users,please use the following command to install English supported version:  
```
bash <(curl -Ls https://raw.githubusercontent.com/torr9522/n5-ui/v0.2.0/install_en.sh)
``` 

The official Stable Runtime is supported on amd64/x86_64 only. ARM64 source compatibility is not an ARM64 Stable Runtime commitment.

## Shortcut  
After Installation，you can input `x-ui`to enter control menu，current menu details：
```
 
  x-ui control menu
  0. exit
————————————————
  1. install   x-ui
  2. update    x-ui
  3. uninstall x-ui
————————————————
  4. reset username
  5. reset panel
  6. reset panel port
  7. check panel info
————————————————
  8. start x-ui
  9. stop  x-ui
  10. restart x-ui
  11. check x-ui status
  12. check x-ui logs
————————————————
  13. enable  x-ui on sysyem startup
  14. disabel x-ui on sysyem startup
————————————————
  15. enable bbr 
  16. issuse certs
 
x-ui status: running
enable on system startup: yes
xray status: running

please input a legal number[0-16]: 
```

# System requirements:  
## MEM  
- 128MB minimal/256MB+ recommend  
## OS
- CentOS 7+
- Ubuntu 16+
- Debian 8+

# Telegram

[Channel](https://t.me/CoderfanBaby)  
[Group](https://t.me/franzkafayu)

# Credits
- [vaxilu/x-ui](https://github.com/vaxilu/x-ui)
- [XTLS/Xray-core](https://github.com/XTLS/Xray-core)
- [telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api)  

# Sponsor  

if you want to purchase some virtual servers,you can purchase by my aff link:   
- [BandwagonHost](https://bandwagonhost.com/aff.php?aff=65703)     
- [Cloudcone](https://app.cloudcone.com/?ref=7536)  
- [SpartanHost](https://billing.spartanhost.net/aff.php?aff=1875)  


## Stargazers over time

[![Stargazers over time](https://starchart.cc/torr9522/n5-ui.svg)](https://starchart.cc/torr9522/n5-ui)
