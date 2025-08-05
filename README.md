# go-magic-image-analysis
wip, with go-scryfall-client

go-cv :

pour dl go cv et la bonne version
git clone le projet
checkout sur la release 0.42.0
dl opencv via le makefile parce que la release package ubuntu est pas à jour avec le package
donc via make install

ensuite vu que y a plein de dependance inutile dans opencv pour go cv

compiler le projet avec go build -tags go_cv_specific_modules pour retirer les modules inutiles type aruco qui ne sont pas utilisés et qui mettent des erreurs de compilation, accessoirement
