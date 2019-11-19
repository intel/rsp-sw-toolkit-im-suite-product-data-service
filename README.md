# Intel® Inventory Suite product-data-service
[![license](https://img.shields.io/badge/license-Apache%20v2.0-blue.svg)](LICENSE)

Product data service is a microservice for the Intel® Inventory Suite and provides persistence and retrieval for enterprise data. 
Enterprise data is ingested through EdgeX platform and consumed by product data service using EdgeX's app functions SDK.
Product data service transforms and persists enterprise data in PostgreSQL and exposes RESTfUl APIs to retrieve data using odata protocol.

# Install and Deploy via Docker Container #

### Prerequisites ###
- Docker & make: 
```
sudo apt install -y docker.io build-essential
```

- Docker-compose:
```
sudo curl \
    -L "https://github.com/docker/compose/releases/download/1.24.0/docker-compose-$(uname -s)-$(uname -m)" \
    -o /usr/local/bin/docker-compose && \
    sudo chmod a+x /usr/local/bin/docker-compose
```

- EdgeX Edingurgh:

```
wget https://raw.githubusercontent.com/edgexfoundry/developer-scripts/master/releases/edinburgh/compose-files/docker-compose-edinburgh-no-secty-1.0.1.yml
sudo docker-compose -f docker-compose-edinburgh-no-secty-1.0.1.yml up -d
```

### Installation ###

```
git clone https://github.impcloud.net/RSP-Inventory-Suite/product-data-service.git
cd product-data-service
sudo make build
sudo make deploy
```

### Incoming Schema ###
```
[
    {
        "exitError":0.0789,
        "becomingReadable":0.0456,
        "beingRead":0,
        "dailyTurn":0.0121,
        "upc":"889319388921",        
        "sku":"11889966"
        "metadata": {
            "color":"red"			
        }
    }
]
```

### Endpoint ###

```
GET http://127.0.0.1:8080/skus
```

#### Results ####
```
{
    "results": [
        {
            "sku": "11889966",
            "productList": [
                {
                    "productId": "889319388921",
                    "beingRead": 0,
                    "becomingReadable": 0.0456,
                    "exitError": 0.0789,
                    "dailyTurn": 0.0121,
                    "metadata": {
                        "color": "red"
                    }
                }
            ]
        }
    ]
}
```

### OData example ###

```
GET http://127.0.0.1:8080/skus?$filter=sku eq '123ABC'
```

#### Results ####
```
{
    "results": [
        {
            "sku": "123ABC",
            "productList": [
                {
                    "productId": "889319388921",
                    "beingRead": 0,
                    "becomingReadable": 0.0456,
                    "exitError": 0.0789,
                    "dailyTurn": 0.0121,
                    "metadata": {
                        "color": "red"
                    }
                }
            ]
        }
    ]
}
```

For more information about odata, visit [OData.org](https://www.odata.org/) and [go-odata](https://github.impcloud.net/RSP-Inventory-Suite/go-odata)