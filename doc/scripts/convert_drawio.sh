#!/bin/bash

# Script to convert Draw.io files to SVG/PNG for MkDocs
# Usage: ./convert_drawio.sh <input.drawio> <output.svg>

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if drawio is installed
if ! command -v drawio &> /dev/null; then
    echo -e "${RED}Error: Draw.io CLI not found${NC}"
    echo "Please install Draw.io Desktop from: https://github.com/jgraph/drawio-desktop/releases"
    echo "Or install via package manager:"
    echo "  - macOS: brew install --cask drawio"
    echo "  - Linux: Download .deb/.rpm from releases page"
    exit 1
fi

# Check arguments
if [ $# -lt 2 ]; then
    echo "Usage: $0 <input.drawio> <output.svg|png>"
    echo "Example: $0 architecture.drawio docs/assets/architecture.svg"
    exit 1
fi

INPUT_FILE="$1"
OUTPUT_FILE="$2"

# Check if input file exists
if [ ! -f "$INPUT_FILE" ]; then
    echo -e "${RED}Error: Input file not found: $INPUT_FILE${NC}"
    exit 1
fi

# Create output directory if it doesn't exist
OUTPUT_DIR=$(dirname "$OUTPUT_FILE")
mkdir -p "$OUTPUT_DIR"

# Determine output format
if [[ "$OUTPUT_FILE" == *.svg ]]; then
    FORMAT="svg"
elif [[ "$OUTPUT_FILE" == *.png ]]; then
    FORMAT="png"
else
    echo -e "${RED}Error: Output format must be .svg or .png${NC}"
    exit 1
fi

echo -e "${YELLOW}Converting $INPUT_FILE to $OUTPUT_FILE...${NC}"

# Convert the file
drawio -x -f "$FORMAT" -o "$OUTPUT_FILE" "$INPUT_FILE"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Successfully converted to $OUTPUT_FILE${NC}"
    
    # Show file size
    if command -v du &> /dev/null; then
        SIZE=$(du -h "$OUTPUT_FILE" | cut -f1)
        echo -e "${GREEN}  File size: $SIZE${NC}"
    fi
else
    echo -e "${RED}✗ Conversion failed${NC}"
    exit 1
fi

# Optimize SVG if svgo is available
if [[ "$FORMAT" == "svg" ]] && command -v svgo &> /dev/null; then
    echo -e "${YELLOW}Optimizing SVG...${NC}"
    svgo "$OUTPUT_FILE" -o "$OUTPUT_FILE"
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ SVG optimized${NC}"
        if command -v du &> /dev/null; then
            SIZE=$(du -h "$OUTPUT_FILE" | cut -f1)
            echo -e "${GREEN}  Optimized size: $SIZE${NC}"
        fi
    fi
fi

# Optimize PNG if optipng is available
if [[ "$FORMAT" == "png" ]] && command -v optipng &> /dev/null; then
    echo -e "${YELLOW}Optimizing PNG...${NC}"
    optipng -quiet "$OUTPUT_FILE"
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ PNG optimized${NC}"
        if command -v du &> /dev/null; then
            SIZE=$(du -h "$OUTPUT_FILE" | cut -f1)
            echo -e "${GREEN}  Optimized size: $SIZE${NC}"
        fi
    fi
fi

echo -e "${GREEN}Done!${NC}"
