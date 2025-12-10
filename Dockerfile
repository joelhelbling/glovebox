FROM ubuntu:24.04

# Avoid interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install base dependencies
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    git \
    build-essential \
    software-properties-common \
    unzip \
    ca-certificates \
    gnupg \
    sudo \
    jq \
    libffi-dev \
    libyaml-dev \
    && rm -rf /var/lib/apt/lists/*

# Install Fish shell
RUN apt-add-repository ppa:fish-shell/release-4 \
    && apt-get update \
    && apt-get install -y fish \
    && rm -rf /var/lib/apt/lists/*

# Install tmux and tmuxp
RUN apt-get update && apt-get install -y \
    tmux \
    python3 \
    python3-pip \
    pipx \
    && rm -rf /var/lib/apt/lists/* \
    && pipx install tmuxp

# Install neovim (latest stable, architecture-aware)
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "amd64" ]; then \
        NVIM_ARCH="linux-x86_64"; \
    elif [ "$ARCH" = "arm64" ]; then \
        NVIM_ARCH="linux-arm64"; \
    else \
        echo "Unsupported architecture: $ARCH" && exit 1; \
    fi && \
    curl -fsSL -o nvim.tar.gz "https://github.com/neovim/neovim/releases/latest/download/nvim-${NVIM_ARCH}.tar.gz" && \
    tar -C /opt -xzf nvim.tar.gz && \
    rm nvim.tar.gz && \
    ln -s /opt/nvim-${NVIM_ARCH}/bin/nvim /usr/local/bin/nvim

# Install Node.js (required for Claude Code and other tools)
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install mise (as root, will be available system-wide)
RUN curl https://mise.run | MISE_INSTALL_PATH=/usr/local/bin/mise sh

# Install Claude Code
RUN npm install -g @anthropic-ai/claude-code

# Install Gemini CLI
RUN npm install -g @google/gemini-cli

# Install OpenCode
RUN npm install -g opencode-ai

# Configure the ubuntu user (UID 1000, already exists in ubuntu:24.04)
RUN usermod -s /usr/bin/fish ubuntu \
    && echo "ubuntu ALL=(root) NOPASSWD:ALL" > /etc/sudoers.d/ubuntu \
    && chmod 0440 /etc/sudoers.d/ubuntu \
    && mkdir -p /home/ubuntu/.local/bin \
    && mkdir -p /home/ubuntu/.local/share/mise \
    && mkdir -p /home/ubuntu/.config/fish \
    && mkdir -p /home/ubuntu/.config/gemini \
    && mkdir -p /home/ubuntu/.anthropic \
    && echo 'mise activate fish | source' >> /home/ubuntu/.config/fish/config.fish \
    && chown -R ubuntu:ubuntu /home/ubuntu

# Copy and set up entrypoint script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod 755 /usr/local/bin/entrypoint.sh

# Switch to non-root user
USER ubuntu
WORKDIR /home/ubuntu

# Ensure pipx and mise binaries are in PATH
ENV PATH="/home/ubuntu/.local/bin:/usr/local/bin:$PATH"

# Set Fish as the default shell
ENV SHELL=/usr/bin/fish

# Set working directory for mounted projects
WORKDIR /workspace

# Use entrypoint to fix permissions on mounted volumes
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["fish"]
