#!/bin/bash
# Test script for opun-nightly Homebrew formula

set -e

echo "🧪 Testing opun-nightly Homebrew formula..."

# Create a test formula file
cat > /tmp/opun-nightly.rb << 'EOF'
class OpunNightly < Formula
  desc "AI code agent automation framework (Nightly/Prerelease)"
  homepage "https://github.com/rizome-dev/opun"
  version "nightly"
  
  # This URL should be replaced with the actual prerelease URL
  url "https://github.com/rizome-dev/opun/releases/download/v1.0.0-rc1/opun_Darwin_x86_64.tar.gz"
  sha256 "REPLACE_WITH_ACTUAL_SHA256"
  
  license "MIT"
  
  conflicts_with "opun", because: "both install the same binaries"
  
  def install
    bin.install "opun"
    
    # Generate and install shell completions
    generate_completions_from_executable(bin/"opun", "completion")
  end
  
  test do
    system "#{bin}/opun", "--version"
  end
  
  def caveats
    <<~EOS
      ⚠️  This is a prerelease version of opun.
      
      For the stable version, use:
        brew install opun
    EOS
  end
end
EOF

echo "📝 Test formula created at /tmp/opun-nightly.rb"
echo ""
echo "To test the formula locally:"
echo "  1. Update the URL and sha256 in the formula"
echo "  2. Run: brew install --build-from-source /tmp/opun-nightly.rb"
echo "  3. Test: opun --version"
echo "  4. Uninstall: brew uninstall opun-nightly"
echo ""
echo "To test with a local build:"
echo "  1. make build"
echo "  2. tar -czf /tmp/opun.tar.gz -C build opun"
echo "  3. Update the formula URL to file:///tmp/opun.tar.gz"
echo "  4. Calculate sha256: shasum -a 256 /tmp/opun.tar.gz"