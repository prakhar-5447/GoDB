# Godb-Server

A lightweight gRPC server built using Golang and packaged as a Docker container.

## Prerequisites
- [Docker](https://www.docker.com/get-started) installed on your system.
- [Git](https://git-scm.com/) installed to clone the repository.

## Running This Project
There are two ways to run this project: either by cloning the repository and running the Go server manually or by pulling the Docker image and running a container. Both methods will run the server on `localhost:50051`, so you can choose either option.

### Option 1: Clone the Repository and Run Manually

```sh
git clone https://github.com/prakhar-5447/godb-server.git
cd godb-server
go run cmd/server/main.go
```

### Option 2: Use Docker

#### Run the Docker Container

To run the container in detached mode and expose the gRPC port **50051**:

```sh
docker run -d -p 50051:50051 --name godb-container prakhar5447/godb-server
```

## Verify Running Server

Check if the server is running:

- **For manual run**: You should see logs in your terminal.
- **For Docker**: Run the following command:

```sh
docker ps
```

If needed, inspect logs:

```sh
docker logs godb-container
```

## Stop and Remove the Docker Container

To stop and remove the container, run:

```sh
docker stop godb-container && docker rm godb-container
```

## Pull from Docker Hub (Alternative)
If you don't want to build the image, you can pull it directly from Docker Hub:

```sh
docker pull prakhar5447/godb-server
```

Then run it as described above.