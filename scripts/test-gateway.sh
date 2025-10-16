#!/bin/bash


echo "🚀 Testing API Gateway Routing"
echo "--------------------------------"

# wait for services to be up
sleep 2

echo "1. Testing Health Check (Direct):"
curl -s http://localhost:8080/health | jq .

echo -e "\n2. Testing User Registration (Direct):"
curl -s -X POST http://localhost:8080/users \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"password123"}' | jq .

echo -e "\n3. Testing Gateway Routing - User Service:"
curl -s http://localhost:8080/api/users/123 | jq .

echo -e "\n4. Testing Gateway Routing - Product Service:"
curl -s http://localhost:8080/api/products/456 | jq .

echo -e "\n5. Testing Unknown Route (Should 404):"
curl -s http://localhost:8080/api/unknown/path | jq .

echo -e "\n6. Testing Load Balancing (Multiple User Requests):"
for i in {1..5}; do
    echo "Request $i:"
    curl -s http://localhost:8080/api/users/test$i | jq -r '.headers["X-Backend-URL"]'
done

echo -e "\n7. Testing Load Balancing (Multiple Product Requests):"
for i in {1..3}; do
    echo "Request $i:"
    curl -s http://localhost:8080/api/products/item$i | jq -r '.headers["X-Backend-URL"]'
done

echo -e "\n✅ Gateway tests completed."