set shell := ["fish", "-c"]

run:
  cd orchestrator/; go run main.go master

update-submodules:
  bash docker-elk/sync_from_upstream.sh 

stop-scanners:
  bash ./stop_scanners.sh

juice-shop:
  echo "juice-shop URL: http://localhost:3000/"
  echo "======================================"
  docker run -p 3000:3000 bkimminich/juice-shop

elk-setup:
  cd docker-elk/; docker compose up setup

elk:
  echo "Kibana URL: http://localhost:5601/"
  echo "username: elastic"
  echo "password: changeme"
  echo "=================================="
  cd docker-elk/; docker compose up
