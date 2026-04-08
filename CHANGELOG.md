# Changelog

## [1.1.0-rc.7](https://github.com/magenx/kuberaptor/compare/v1.0.1-rc.7...v1.1.0-rc.7) (2026-04-08)


### 🚀 Features

* Add Flux CLI tool and --skip-tools flag to create command ([#118](https://github.com/magenx/kuberaptor/issues/118)) ([3893060](https://github.com/magenx/kuberaptor/commit/3893060ea44afaee692bea08f8faa0fb4b5aed4d))
* Add Kured (Kubernetes Reboot Daemon) addon ([#119](https://github.com/magenx/kuberaptor/issues/119)) ([09e7a5c](https://github.com/magenx/kuberaptor/commit/09e7a5c2becba4b5dd650b172e4276339edc10e6))
* Apply Hetzner delete protection to servers, load balancers, and networks ([#115](https://github.com/magenx/kuberaptor/issues/115)) ([03ea26f](https://github.com/magenx/kuberaptor/commit/03ea26f49faf48b3fbe1d77b9636412194de83c5))


### 🛠️ Refactoring

* Tools installation with brew (macOS) and winget (Windows) ([#114](https://github.com/magenx/kuberaptor/issues/114)) ([2f00162](https://github.com/magenx/kuberaptor/commit/2f001623932eccef39e61c106c00464616128643))


### 🚦 Maintenance

* Update go fmt command to include './' prefix ([#120](https://github.com/magenx/kuberaptor/issues/120)) ([9d1d77d](https://github.com/magenx/kuberaptor/commit/9d1d77d74830997961bfb3f7a8fa35afd4f25305))
* **workflow:** Fix typo in release body upload step ([#111](https://github.com/magenx/kuberaptor/issues/111)) ([e3ed36e](https://github.com/magenx/kuberaptor/commit/e3ed36edde292001928c2ee7492ecc9a2155d95b))

## [1.0.1-rc.7](https://github.com/magenx/kuberaptor/compare/v1.0.1-rc.6...v1.0.1-rc.7) (2026-04-07)


### 🐛 Bug Fixes

* Update curl command options for downloading tarball ([#109](https://github.com/magenx/kuberaptor/issues/109)) ([fc689f4](https://github.com/magenx/kuberaptor/commit/fc689f41b20416c3d91058dad4ef15de99feec19))


### 🚦 Maintenance

* **workflow:** Enhance build workflow with checksum generation ([#108](https://github.com/magenx/kuberaptor/issues/108)) ([55b6cba](https://github.com/magenx/kuberaptor/commit/55b6cbacaa0f888d802d4852804950a2584df34f))

## [1.0.1-rc.6](https://github.com/magenx/kuberaptor/compare/v1.0.1-rc.5...v1.0.1-rc.6) (2026-04-06)


### 🛠️ Refactoring

* Remove dead code - unused structs, empty vars, config field ([65b13ae](https://github.com/magenx/kuberaptor/commit/65b13aedcedcfe556aebb7116d1c0ec980752a6f))

## [1.0.1-rc.5](https://github.com/magenx/kuberaptor/compare/v1.0.1-rc.4...v1.0.1-rc.5) (2026-04-06)


### 🛠️ Refactoring

* Update curl options for downloading tools ([#105](https://github.com/magenx/kuberaptor/issues/105)) ([33e0fb3](https://github.com/magenx/kuberaptor/commit/33e0fb3320d12dd4b5f464df9f358f1e19aa4f35))

## [1.0.1-rc.4](https://github.com/magenx/kuberaptor/compare/v1.0.1-rc.3...v1.0.1-rc.4) (2026-04-06)


### 🐛 Bug Fixes

* update copilot instructions to match current codebase state ([f293a2a](https://github.com/magenx/kuberaptor/commit/f293a2a51e957fdd3e5d2aa0f1c4b658888d0bd8))


### 🛠️ Refactoring

* Refactor SSH configuration and service management ([#103](https://github.com/magenx/kuberaptor/issues/103)) ([56968f2](https://github.com/magenx/kuberaptor/commit/56968f20f3b59a237f4d803cf722a50bca4c9c76))


### 📝 Documentation

* **copilot:** Sync copilot instructions with current codebase state ([#102](https://github.com/magenx/kuberaptor/issues/102)) ([f293a2a](https://github.com/magenx/kuberaptor/commit/f293a2a51e957fdd3e5d2aa0f1c4b658888d0bd8))

## [1.0.1-rc.3](https://github.com/magenx/kuberaptor/compare/v1.0.1-rc.2...v1.0.1-rc.3) (2026-04-05)


### 📝 Documentation

* **readme:** Add features comparison table to README ([#99](https://github.com/magenx/kuberaptor/issues/99)) ([85c788a](https://github.com/magenx/kuberaptor/commit/85c788ae1e582312bf4988a48fba9de9050cde26))


### 🚦 Maintenance

* Add copyright notice for 2023 Vito Botta ([#100](https://github.com/magenx/kuberaptor/issues/100)) ([4f139df](https://github.com/magenx/kuberaptor/commit/4f139df42a5c4557d6fafa05b692ba4de9c74b4e))
* **readme:** Revise README with new documentation and formatting ([#98](https://github.com/magenx/kuberaptor/issues/98)) ([06b6a76](https://github.com/magenx/kuberaptor/commit/06b6a7605118fb8efe1555ae4bce5804fc249e0a))

## [1.0.1-rc.2](https://github.com/magenx/kuberaptor/compare/v1.0.1-rc.1...v1.0.1-rc.2) (2026-04-05)


### 🐛 Bug Fixes

* Add `preserve` option to DNS zone to prevent deletion on cluster teardown ([#96](https://github.com/magenx/kuberaptor/issues/96)) ([2f153ad](https://github.com/magenx/kuberaptor/commit/2f153ad2cbcafd887b3eda72cf4f0579fc637f01))
* Pass correct SSH key name to cluster autoscaler ([#94](https://github.com/magenx/kuberaptor/issues/94)) ([ed4e4e0](https://github.com/magenx/kuberaptor/commit/ed4e4e0daf853a0d9f232ae0c82e54d3be95e286))


### 📝 Documentation

* **readme:** Add DNS preconfiguration for certificate validation ([#92](https://github.com/magenx/kuberaptor/issues/92)) ([b067668](https://github.com/magenx/kuberaptor/commit/b067668ca52ade46795fb1438d4d621d8084ba36))
* **readme:** Enhance cluster configuration documentation ([#91](https://github.com/magenx/kuberaptor/issues/91)) ([fe9bb7d](https://github.com/magenx/kuberaptor/commit/fe9bb7d7d63149e8a149ee1515d9fccef3e30b66))
* **readme:** Replace external image links with local screenshots ([#90](https://github.com/magenx/kuberaptor/issues/90)) ([852667a](https://github.com/magenx/kuberaptor/commit/852667a1a83a9e96cdfba62efd61311ffe507533))
* **readme:** Update live site URL in README.md ([#87](https://github.com/magenx/kuberaptor/issues/87)) ([b76a387](https://github.com/magenx/kuberaptor/commit/b76a387ec5a209e96e0ed4bb36673a6807fe4a82))


### 🚦 Maintenance

* Create README.md ([#88](https://github.com/magenx/kuberaptor/issues/88)) ([04b7f38](https://github.com/magenx/kuberaptor/commit/04b7f38b02ffd494bf8301171e992e2c239ea369))
* Kuberaptor cli images ([#89](https://github.com/magenx/kuberaptor/issues/89)) ([864a63b](https://github.com/magenx/kuberaptor/commit/864a63bab02328d01e24062f923fae8fd3ab4d64))

## [1.0.1-rc.1](https://github.com/magenx/kuberaptor/compare/v1.0.1-rc...v1.0.1-rc.1) (2026-04-04)


### 🚦 Maintenance

* Fix comment grammar in shell_test.go ([#76](https://github.com/magenx/kuberaptor/issues/76)) ([ee0a08a](https://github.com/magenx/kuberaptor/commit/ee0a08a5db2d73cbce75016f63212c88744e3b64))
* Fix comment grammar in shell_test.go ([#78](https://github.com/magenx/kuberaptor/issues/78)) ([d76f011](https://github.com/magenx/kuberaptor/commit/d76f011851bcb27ee3878d6885b86b2f2588a700))
* Fix comment grammar in shell_test.go ([#85](https://github.com/magenx/kuberaptor/issues/85)) ([94c1d3a](https://github.com/magenx/kuberaptor/commit/94c1d3a1ad84c4b93739a9f1e0c0259c8be9fd6a))
* **main:** kuberaptor 1.0.0 ([#79](https://github.com/magenx/kuberaptor/issues/79)) ([103076a](https://github.com/magenx/kuberaptor/commit/103076a51869bc38441bb32de4d338282958d7ae))
* **main:** kuberaptor 1.0.1-rc ([#80](https://github.com/magenx/kuberaptor/issues/80)) ([5c3d2e2](https://github.com/magenx/kuberaptor/commit/5c3d2e2da1a70f30e0bd956308b3dd3c3d4acf1c))
* Update download URL for binary in release workflow ([#84](https://github.com/magenx/kuberaptor/issues/84)) ([fef6ddb](https://github.com/magenx/kuberaptor/commit/fef6ddbf2f6ac0c616840c4dae1edd08293ea994))
* Update project website link and add LinkedIn reference ([#82](https://github.com/magenx/kuberaptor/issues/82)) ([42e59c1](https://github.com/magenx/kuberaptor/commit/42e59c118bf564a70945b5ccd3f26164b0a800cc))
* Update version fetching method in build workflow ([#83](https://github.com/magenx/kuberaptor/issues/83)) ([905f22a](https://github.com/magenx/kuberaptor/commit/905f22a3890adbc5545ccef5222fc91bff896a02))

## [1.0.1-rc](https://github.com/magenx/kuberaptor/compare/v1.0.0...v1.0.1-rc) (2026-03-31)


### 🚦 Maintenance

* Fix comment grammar in shell_test.go ([#76](https://github.com/magenx/kuberaptor/issues/76)) ([ee0a08a](https://github.com/magenx/kuberaptor/commit/ee0a08a5db2d73cbce75016f63212c88744e3b64))
* Fix comment grammar in shell_test.go ([#78](https://github.com/magenx/kuberaptor/issues/78)) ([d76f011](https://github.com/magenx/kuberaptor/commit/d76f011851bcb27ee3878d6885b86b2f2588a700))
* **main:** kuberaptor 1.0.0 ([#79](https://github.com/magenx/kuberaptor/issues/79)) ([103076a](https://github.com/magenx/kuberaptor/commit/103076a51869bc38441bb32de4d338282958d7ae))

## 1.0.0 (2026-03-31)


### 🚦 Maintenance

* Fix comment grammar in shell_test.go ([#76](https://github.com/magenx/kuberaptor/issues/76)) ([ee0a08a](https://github.com/magenx/kuberaptor/commit/ee0a08a5db2d73cbce75016f63212c88744e3b64))
* Fix comment grammar in shell_test.go ([#78](https://github.com/magenx/kuberaptor/issues/78)) ([d76f011](https://github.com/magenx/kuberaptor/commit/d76f011851bcb27ee3878d6885b86b2f2588a700))
