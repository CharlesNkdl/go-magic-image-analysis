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

pour gotesseract , ne pas oublier de sudo apt install libtesseract-dev sinon pb de compilation sur un manque de .h

pour test de processing d'image meme si maintenant c'est que des images clean
go test -v ./processing/

go run main.go
pour lancer le serveur gin

curl -X POST \
  http://localhost:8080/analyze \
  -F "cardImage=@PATH/go-magic-image-analysis/test_images/sol_ring.jpg" \
  -H "Content-Type: multipart/form-data"

first step done
