generate-key: # Generate a new key pair for testing
	rm -f ./test-key*
	ssh-keygen -q -f ./test-key -t ecdsa -N ''
	cp ./test-key.pub ./sshd/authorized_keys

up: generate-key # Start the testing environment
	docker compose up -d --build --wait
