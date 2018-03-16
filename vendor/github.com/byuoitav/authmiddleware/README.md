# authmiddleware
[![CircleCI](https://img.shields.io/circleci/project/byuoitav/authmiddleware.svg)](https://circleci.com/gh/byuoitav/authmiddleware) [![Apache 2 License](https://img.shields.io/hexpm/l/plug.svg)](https://raw.githubusercontent.com/byuoitav/authmiddleware/master/LICENSE)

Validating [JWT tokens](https://jwt.io/) with BYU's implementation of [WSO2](http://wso2.com/products/api-manager/) isn't the easiest thing on the planet. We wanted to make sure that we protected every route with a JWT check and, as part of that check, pull in the latest signing key.

## Installation
```
go get github.com/byuoitav/authmiddleware
```

## Example Usage
```
import "github.com/byuoitav/authmiddleware"
```
```
func main() {
	router := echo.New()
	router.Use(echo.WrapMiddleware(authmiddleware.ValidateJWT))

	router.Start()
}
```

## Local Development
The WSO2 JWT checking "pipeline" will not work from your local box. To enable local development without headaches, be sure to set the `LOCAL_ENVIRONMENT` variable in you shell environment.

Example (using `/etc/environment`):
```
export LOCAL_ENVIRONMENT="true"
```
