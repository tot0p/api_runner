# api_runner

api_runner is project linked to [runner](https://github.com/tot0p/runner) project. It is a simple API to manage docker containers.

## docs of the API

| Method  |       Route        |                Description                 |
|:-------:|:------------------:|:------------------------------------------:|
|   GET   |        /vm         |             Get all containers             |
|   GET   | /vm/:id/containers |       Get container by container id        |
|   GET   |       /ping        |             Get ping response              |
|   GET   |       /logs        | Get logs of containers creation & deletion |
|  POST   |        /vm         |             Create a container             |
|  POST   |       /build       |      Build an image from a Dockerfile      |
| DELETE  |      /vm/:id       |      Delete container by container id      |

### Create a container: (POST /vm)
```json
{
  "link": "https://github.com/username/repo"
}
```

### Build an image from a Dockerfile: (POST /build)
```json
{
  "repository": "https://github.com/username/repo"
}
```

## License

[MIT](https://choosealicense.com/licenses/mit/)

[![](https://contributors-img.web.app/image?repo=tot0p/api_runner)](https://github.com/tot0p/api_runner/graphs/contributors)