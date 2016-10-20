# api

## multile regions:

GET /signin -H 'Authorization: Basic xxx'

[
{
    "access_token": "xsdxdasfaswqewqe",
        "expires_in": "86400",
        "token_type": "Bearer",
        "region": "cn-north-2"

},
{
    "access_token": "wefqqweda",
    "expires_in": "86400",
    "token_type": "Bearer",
    "region": "cn-north-1"

}
]

## login

GET /login -H 'Authorization: Basic xxx'

{"access_token":"asddfed","expires_in":"86400","token_type":"Bearer"}
