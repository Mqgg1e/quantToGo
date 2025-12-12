#!/bin/bash
# Test UserDataStream functionality

echo "🧪 Testing UserDataStream..."
echo ""

cd "$(dirname "$0")/.." || exit 1

# Build
echo "Building test program..."
go build -o bin/test-userdata-stream cmd/test-userdata-stream/main.go

if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful"
echo ""

# Run
echo "Running test..."
echo "Press Ctrl+C to stop"
echo ""

./bin/test-userdata-stream

