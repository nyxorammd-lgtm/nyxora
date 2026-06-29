# typed: true
# frozen_string_literal: true

class Nyxora < Formula
  desc "Adaptive Tunnel Orchestrator — self-healing multi-transport VPN/tunnel manager"
  homepage "https://github.com/nyxorammd-lgtm/nyxora"
  version "0.2.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/nyxorammd-lgtm/nyxora/releases/download/v0.2.0/nyxora_darwin_amd64"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000" # placeholder
    end
    if Hardware::CPU.arm?
      url "https://github.com/nyxorammd-lgtm/nyxora/releases/download/v0.2.0/nyxora_darwin_arm64"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000" # placeholder
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/nyxorammd-lgtm/nyxora/releases/download/v0.2.0/nyxora_linux_amd64"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000" # placeholder
    end
    if Hardware::CPU.arm?
      url "https://github.com/nyxorammd-lgtm/nyxora/releases/download/v0.2.0/nyxora_linux_arm64"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000" # placeholder
    end
  end

  def install
    bin.install "nyxora"
  end

  test do
    assert_match "v#{version}", shell_output("#{bin}/nyxora version")
  end
end
