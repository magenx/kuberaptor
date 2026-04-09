# Changelog

## [1.1.0-rc.13](https://github.com/magenx/kuberaptor/compare/v1.1.0-rc.12...v1.1.0-rc.13) (2026-04-09)


### 🚀 Features

* Add Flux CLI tool and --skip-tools flag to create command ([#118](https://github.com/magenx/kuberaptor/issues/118)) ([3893060](https://github.com/magenx/kuberaptor/commit/3893060ea44afaee692bea08f8faa0fb4b5aed4d))
* Add Kured (Kubernetes Reboot Daemon) addon ([#119](https://github.com/magenx/kuberaptor/issues/119)) ([09e7a5c](https://github.com/magenx/kuberaptor/commit/09e7a5c2becba4b5dd650b172e4276339edc10e6))
* Apply Hetzner delete protection to servers, load balancers, and networks ([#115](https://github.com/magenx/kuberaptor/issues/115)) ([03ea26f](https://github.com/magenx/kuberaptor/commit/03ea26f49faf48b3fbe1d77b9636412194de83c5))


### 🐛 Bug Fixes

* Add `preserve` option to DNS zone to prevent deletion on cluster teardown ([#96](https://github.com/magenx/kuberaptor/issues/96)) ([2f153ad](https://github.com/magenx/kuberaptor/commit/2f153ad2cbcafd887b3eda72cf4f0579fc637f01))
* Add rebuild protection option to ChangeServerProtection ([#132](https://github.com/magenx/kuberaptor/issues/132)) ([c226a0f](https://github.com/magenx/kuberaptor/commit/c226a0fb147afeada66c0b165034c47da80bfaaf))
* Pass correct SSH key name to cluster autoscaler ([#94](https://github.com/magenx/kuberaptor/issues/94)) ([ed4e4e0](https://github.com/magenx/kuberaptor/commit/ed4e4e0daf853a0d9f232ae0c82e54d3be95e286))
* update copilot instructions to match current codebase state ([f293a2a](https://github.com/magenx/kuberaptor/commit/f293a2a51e957fdd3e5d2aa0f1c4b658888d0bd8))
* Update curl command options for downloading tarball ([#109](https://github.com/magenx/kuberaptor/issues/109)) ([fc689f4](https://github.com/magenx/kuberaptor/commit/fc689f41b20416c3d91058dad4ef15de99feec19))


### 🛠️ Refactoring

* Optimizing code and installation for usability ([#138](https://github.com/magenx/kuberaptor/issues/138)) ([96dc069](https://github.com/magenx/kuberaptor/commit/96dc0695c5b782b0b764ecfb79111f7a1bdd9c4c))
* Refactor SSH configuration and service management ([#103](https://github.com/magenx/kuberaptor/issues/103)) ([56968f2](https://github.com/magenx/kuberaptor/commit/56968f20f3b59a237f4d803cf722a50bca4c9c76))
* Remove dead code - unused structs, empty vars, config field ([65b13ae](https://github.com/magenx/kuberaptor/commit/65b13aedcedcfe556aebb7116d1c0ec980752a6f))
* Support Homebrew installation on Linux ([#136](https://github.com/magenx/kuberaptor/issues/136)) ([396e0c0](https://github.com/magenx/kuberaptor/commit/396e0c016f89bf531cc1667d982c18f3f7370851))
* Tools installation with brew (macOS) and winget (Windows) ([#114](https://github.com/magenx/kuberaptor/issues/114)) ([2f00162](https://github.com/magenx/kuberaptor/commit/2f001623932eccef39e61c106c00464616128643))
* Update curl options for downloading tools ([#105](https://github.com/magenx/kuberaptor/issues/105)) ([33e0fb3](https://github.com/magenx/kuberaptor/commit/33e0fb3320d12dd4b5f464df9f358f1e19aa4f35))


### 📝 Documentation

* **copilot:** Sync copilot instructions with current codebase state ([#102](https://github.com/magenx/kuberaptor/issues/102)) ([f293a2a](https://github.com/magenx/kuberaptor/commit/f293a2a51e957fdd3e5d2aa0f1c4b658888d0bd8))
* **readme:** Add DNS preconfiguration for certificate validation ([#92](https://github.com/magenx/kuberaptor/issues/92)) ([b067668](https://github.com/magenx/kuberaptor/commit/b067668ca52ade46795fb1438d4d621d8084ba36))
* **readme:** Add features comparison table to README ([#99](https://github.com/magenx/kuberaptor/issues/99)) ([85c788a](https://github.com/magenx/kuberaptor/commit/85c788ae1e582312bf4988a48fba9de9050cde26))
* **readme:** Enhance cluster configuration documentation ([#91](https://github.com/magenx/kuberaptor/issues/91)) ([fe9bb7d](https://github.com/magenx/kuberaptor/commit/fe9bb7d7d63149e8a149ee1515d9fccef3e30b66))
* **readme:** Replace external image links with local screenshots ([#90](https://github.com/magenx/kuberaptor/issues/90)) ([852667a](https://github.com/magenx/kuberaptor/commit/852667a1a83a9e96cdfba62efd61311ffe507533))
* **readme:** Update live site URL in README.md ([#87](https://github.com/magenx/kuberaptor/issues/87)) ([b76a387](https://github.com/magenx/kuberaptor/commit/b76a387ec5a209e96e0ed4bb36673a6807fe4a82))


### 🚦 Maintenance

* Add copyright notice for 2023 Vito Botta ([#100](https://github.com/magenx/kuberaptor/issues/100)) ([4f139df](https://github.com/magenx/kuberaptor/commit/4f139df42a5c4557d6fafa05b692ba4de9c74b4e))
* Create README.md ([#88](https://github.com/magenx/kuberaptor/issues/88)) ([04b7f38](https://github.com/magenx/kuberaptor/commit/04b7f38b02ffd494bf8301171e992e2c239ea369))
* Enhance README with Hetzner links and details ([#134](https://github.com/magenx/kuberaptor/issues/134)) ([7c34bce](https://github.com/magenx/kuberaptor/commit/7c34bcef5eb0b223116e0e3835aeb3d374cda79a))
* Fix comment grammar in shell_test.go ([#126](https://github.com/magenx/kuberaptor/issues/126)) ([ddc8c82](https://github.com/magenx/kuberaptor/commit/ddc8c82655748d46fbcea5fac6cbb2d013f9f105))
* Fix comment grammar in shell_test.go ([#130](https://github.com/magenx/kuberaptor/issues/130)) ([4a26d1c](https://github.com/magenx/kuberaptor/commit/4a26d1cf9ce7406eb2b06113d9f7aaa3041799ca))
* Fix comment grammar in shell_test.go ([#76](https://github.com/magenx/kuberaptor/issues/76)) ([ee0a08a](https://github.com/magenx/kuberaptor/commit/ee0a08a5db2d73cbce75016f63212c88744e3b64))
* Fix comment grammar in shell_test.go ([#78](https://github.com/magenx/kuberaptor/issues/78)) ([d76f011](https://github.com/magenx/kuberaptor/commit/d76f011851bcb27ee3878d6885b86b2f2588a700))
* Fix comment grammar in shell_test.go ([#85](https://github.com/magenx/kuberaptor/issues/85)) ([94c1d3a](https://github.com/magenx/kuberaptor/commit/94c1d3a1ad84c4b93739a9f1e0c0259c8be9fd6a))
* Fix formatting and remove unnecessary newlines ([#125](https://github.com/magenx/kuberaptor/issues/125)) ([c2b3100](https://github.com/magenx/kuberaptor/commit/c2b3100678718fd2f8b96e441792d2d261b55474))
* Kuberaptor cli images ([#89](https://github.com/magenx/kuberaptor/issues/89)) ([864a63b](https://github.com/magenx/kuberaptor/commit/864a63bab02328d01e24062f923fae8fd3ab4d64))
* **main:** kuberaptor 1.0.0 ([#79](https://github.com/magenx/kuberaptor/issues/79)) ([103076a](https://github.com/magenx/kuberaptor/commit/103076a51869bc38441bb32de4d338282958d7ae))
* **main:** kuberaptor 1.0.1-rc ([#80](https://github.com/magenx/kuberaptor/issues/80)) ([5c3d2e2](https://github.com/magenx/kuberaptor/commit/5c3d2e2da1a70f30e0bd956308b3dd3c3d4acf1c))
* **main:** kuberaptor 1.0.1-rc.1 ([#81](https://github.com/magenx/kuberaptor/issues/81)) ([92c38db](https://github.com/magenx/kuberaptor/commit/92c38db3589d8b3876274c2b14cc44ba9bd2cf09))
* **main:** kuberaptor 1.0.1-rc.2 ([#86](https://github.com/magenx/kuberaptor/issues/86)) ([c6c19b3](https://github.com/magenx/kuberaptor/commit/c6c19b3be9ed86e5765e144b643efc86c3dbc0be))
* **main:** kuberaptor 1.0.1-rc.3 ([#97](https://github.com/magenx/kuberaptor/issues/97)) ([95ff443](https://github.com/magenx/kuberaptor/commit/95ff443b67926de510c5b83f0204c333438028d6))
* **main:** kuberaptor 1.0.1-rc.4 ([#101](https://github.com/magenx/kuberaptor/issues/101)) ([391d3bd](https://github.com/magenx/kuberaptor/commit/391d3bdb255e44e064800d837f0361af4b6fefdd))
* **main:** kuberaptor 1.0.1-rc.5 ([#104](https://github.com/magenx/kuberaptor/issues/104)) ([a651131](https://github.com/magenx/kuberaptor/commit/a65113171c98add75263e991e09b7d5fffd05517))
* **main:** kuberaptor 1.0.1-rc.6 ([#106](https://github.com/magenx/kuberaptor/issues/106)) ([270d601](https://github.com/magenx/kuberaptor/commit/270d601e72ef54873a2ca674b0660a20cd7096cb))
* **main:** kuberaptor 1.0.1-rc.7 ([#107](https://github.com/magenx/kuberaptor/issues/107)) ([afe7b95](https://github.com/magenx/kuberaptor/commit/afe7b95a20b3a753d9806d5a694ade6c6f5c3fd3))
* **main:** kuberaptor 1.1.0-rc.10 ([#127](https://github.com/magenx/kuberaptor/issues/127)) ([513c47f](https://github.com/magenx/kuberaptor/commit/513c47f070b2cd26365030b42c168defad46d833))
* **main:** kuberaptor 1.1.0-rc.11 ([#131](https://github.com/magenx/kuberaptor/issues/131)) ([5079f6b](https://github.com/magenx/kuberaptor/commit/5079f6b18e7c83ac19d4fce7a30a9e9a93229afd))
* **main:** kuberaptor 1.1.0-rc.12 ([#133](https://github.com/magenx/kuberaptor/issues/133)) ([0e74ec7](https://github.com/magenx/kuberaptor/commit/0e74ec767fd842c530f9650b171f3b7b21580dab))
* **main:** kuberaptor 1.1.0-rc.7 ([#110](https://github.com/magenx/kuberaptor/issues/110)) ([8a049d5](https://github.com/magenx/kuberaptor/commit/8a049d5083c10bb5b1a83fb44b6e0b52c3ce2629))
* **main:** kuberaptor 1.1.0-rc.8 ([#121](https://github.com/magenx/kuberaptor/issues/121)) ([83f920a](https://github.com/magenx/kuberaptor/commit/83f920a3688b7af3c8416b85097cfd711a925cd0))
* **main:** kuberaptor 1.1.0-rc.9 ([#124](https://github.com/magenx/kuberaptor/issues/124)) ([a2096cd](https://github.com/magenx/kuberaptor/commit/a2096cddbf0ac42e677f2259ea7a3f09b337fa81))
* **readme:** Revise README with new documentation and formatting ([#98](https://github.com/magenx/kuberaptor/issues/98)) ([06b6a76](https://github.com/magenx/kuberaptor/commit/06b6a7605118fb8efe1555ae4bce5804fc249e0a))
* Test release build and workflow ([#123](https://github.com/magenx/kuberaptor/issues/123)) ([c72c912](https://github.com/magenx/kuberaptor/commit/c72c91281575daa9ae81d25b673b539cd08d196b))
* Update download URL for binary in release workflow ([#84](https://github.com/magenx/kuberaptor/issues/84)) ([fef6ddb](https://github.com/magenx/kuberaptor/commit/fef6ddbf2f6ac0c616840c4dae1edd08293ea994))
* Update go fmt command to include './' prefix ([#120](https://github.com/magenx/kuberaptor/issues/120)) ([9d1d77d](https://github.com/magenx/kuberaptor/commit/9d1d77d74830997961bfb3f7a8fa35afd4f25305))
* Update installation commands with repository name ([#128](https://github.com/magenx/kuberaptor/issues/128)) ([ff974bb](https://github.com/magenx/kuberaptor/commit/ff974bb1990b803c3575cf7012afe089fd860c71))
* Update project website link and add LinkedIn reference ([#82](https://github.com/magenx/kuberaptor/issues/82)) ([42e59c1](https://github.com/magenx/kuberaptor/commit/42e59c118bf564a70945b5ccd3f26164b0a800cc))
* Update version fetching method in build workflow ([#83](https://github.com/magenx/kuberaptor/issues/83)) ([905f22a](https://github.com/magenx/kuberaptor/commit/905f22a3890adbc5545ccef5222fc91bff896a02))
* **workflow:** Add Windows build support to CI workflow ([#135](https://github.com/magenx/kuberaptor/issues/135)) ([040002b](https://github.com/magenx/kuberaptor/commit/040002becf99f77885cf8e81adf02ed99566f753))
* **workflow:** Enhance build workflow with checksum generation ([#108](https://github.com/magenx/kuberaptor/issues/108)) ([55b6cba](https://github.com/magenx/kuberaptor/commit/55b6cbacaa0f888d802d4852804950a2584df34f))
* **workflow:** Fix typo in release body upload step ([#111](https://github.com/magenx/kuberaptor/issues/111)) ([e3ed36e](https://github.com/magenx/kuberaptor/commit/e3ed36edde292001928c2ee7492ecc9a2155d95b))
* **workflow:** Update installation commands for repository name ([#122](https://github.com/magenx/kuberaptor/issues/122)) ([eaa05f9](https://github.com/magenx/kuberaptor/commit/eaa05f9789d738a1dd84231d2217e5c30fc50ed6))
* **wrkflow:** Downgrade action-gh-release version to v2.5.0 ([#129](https://github.com/magenx/kuberaptor/issues/129)) ([babe6d5](https://github.com/magenx/kuberaptor/commit/babe6d5477ff8656f6257a473fec624d8e698312))

## [1.1.0-rc.12](https://github.com/magenx/kuberaptor/compare/v1.1.0-rc.11...v1.1.0-rc.12) (2026-04-09)


### 🛠️ Refactoring

* Support Homebrew installation on Linux ([#136](https://github.com/magenx/kuberaptor/issues/136)) ([396e0c0](https://github.com/magenx/kuberaptor/commit/396e0c016f89bf531cc1667d982c18f3f7370851))


### 🚦 Maintenance

* Enhance README with Hetzner links and details ([#134](https://github.com/magenx/kuberaptor/issues/134)) ([7c34bce](https://github.com/magenx/kuberaptor/commit/7c34bcef5eb0b223116e0e3835aeb3d374cda79a))
* **workflow:** Add Windows build support to CI workflow ([#135](https://github.com/magenx/kuberaptor/issues/135)) ([040002b](https://github.com/magenx/kuberaptor/commit/040002becf99f77885cf8e81adf02ed99566f753))

## [1.1.0-rc.11](https://github.com/magenx/kuberaptor/compare/v1.1.0-rc.10...v1.1.0-rc.11) (2026-04-08)


### 🐛 Bug Fixes

* Add rebuild protection option to ChangeServerProtection ([#132](https://github.com/magenx/kuberaptor/issues/132)) ([c226a0f](https://github.com/magenx/kuberaptor/commit/c226a0fb147afeada66c0b165034c47da80bfaaf))

## [1.1.0-rc.10](https://github.com/magenx/kuberaptor/compare/v1.1.0-rc.9...v1.1.0-rc.10) (2026-04-08)


### 🚦 Maintenance

* Fix comment grammar in shell_test.go ([#130](https://github.com/magenx/kuberaptor/issues/130)) ([4a26d1c](https://github.com/magenx/kuberaptor/commit/4a26d1cf9ce7406eb2b06113d9f7aaa3041799ca))
* Update installation commands with repository name ([#128](https://github.com/magenx/kuberaptor/issues/128)) ([ff974bb](https://github.com/magenx/kuberaptor/commit/ff974bb1990b803c3575cf7012afe089fd860c71))
* **wrkflow:** Downgrade action-gh-release version to v2.5.0 ([#129](https://github.com/magenx/kuberaptor/issues/129)) ([babe6d5](https://github.com/magenx/kuberaptor/commit/babe6d5477ff8656f6257a473fec624d8e698312))

## [1.1.0-rc.9](https://github.com/magenx/kuberaptor/compare/v1.1.0-rc.8...v1.1.0-rc.9) (2026-04-08)


### 🚦 Maintenance

* Fix comment grammar in shell_test.go ([#126](https://github.com/magenx/kuberaptor/issues/126)) ([ddc8c82](https://github.com/magenx/kuberaptor/commit/ddc8c82655748d46fbcea5fac6cbb2d013f9f105))
* Fix formatting and remove unnecessary newlines ([#125](https://github.com/magenx/kuberaptor/issues/125)) ([c2b3100](https://github.com/magenx/kuberaptor/commit/c2b3100678718fd2f8b96e441792d2d261b55474))

## [1.1.0-rc.8](https://github.com/magenx/kuberaptor/compare/v1.1.0-rc.7...v1.1.0-rc.8) (2026-04-08)


### 🚦 Maintenance

* Test release build and workflow ([#123](https://github.com/magenx/kuberaptor/issues/123)) ([c72c912](https://github.com/magenx/kuberaptor/commit/c72c91281575daa9ae81d25b673b539cd08d196b))
* **workflow:** Update installation commands for repository name ([#122](https://github.com/magenx/kuberaptor/issues/122)) ([eaa05f9](https://github.com/magenx/kuberaptor/commit/eaa05f9789d738a1dd84231d2217e5c30fc50ed6))

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
