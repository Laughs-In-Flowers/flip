## Changelog

### flip 0.1.1 (12.11.2019)
- smooth over interface surfaces 
- internal execution refactor to rank commands more effectively
- reduce sort.Sort cruft


### flip 0.1.0 (03.04.2019)
- eliminate external data package dependency(except for testing) in preference  to specific interfaces
- nomenclature changes, reversions, and sorting
- version bump up one level
- examples in README
- added TODO


### flip 0.0.4 (22.08.2018)
- changes to api & nomenclature to accomodate external use & configuration


### flip 0.0.3 (19.06.2018)
- testing & associated finetuneing
- RegexVar & RegexVectorVar flag functionality


### flip 0.0.2 (22.11.2017)
- continued rewriting & refactoring
- specific interfaces for managing core function, instructions, execution, and cleanup
- command fall-though to stop processing (useful for help command to capture params and not process as a command)
- improved instruction formatting
- added user defined post command functionality by SetCleanup
- integration of flagsets & flags using data.Vector keys & values
- api for adding builtin commands(i.e. help, versioning, and future additions)
- builtin Help and Version commands


### flip 0.0.1 (20.09.2016)
- initialize public package 
