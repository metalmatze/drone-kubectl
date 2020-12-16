## 0.3.0 / 2020-12-16

Bring the entire project from 2017 to 2020

* [ENHANCEMENT] Update kubectl to v1.19.4
* [ENHANCEMENT] Update to Go 1.15
* [FIX] Fix this plugin to work with new Drone config - needed to add prefix `PLUGIN_` to `PLUGIN_KUBECONFIG`  

## 0.2.2 / 2018-10-15
* [ENHANCEMENT] Update kubectl to v1.12.1
* [FIX] Using in cluster credentials for kubectl correctly.
* [FEATURE] Add debug mode.

## 0.2.1 / 2018-05-29

* [ENHANCEMENT] Update kubectl to v1.10.3

## 0.2.0 / 2018-02-01

* [FEATURE] Allow kubectl commands to use templating [#5]
* [ENHANCEMENT] Update kubectl to v1.9.2

## 0.1.0 / 2018-01-28

Initial Release

* [FEATURE] Add string helpers like b64enc, trunc, trim, upper and lower [#4]
* [ENHANCEMENT] Pass the complete environment to templates [#2]
