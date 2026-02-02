export interface TechTag {
  readonly name: string;
  readonly color: string;
}

export function getRuntimeTag(runtime: string): TechTag | null {
  const tags: Record<string, TechTag> = {
    go: {
      name: "Go",
      color: "bg-cyan-500/20 text-cyan-400 border-cyan-500/30",
    },
    node: {
      name: "Node.js",
      color: "bg-green-500/20 text-green-400 border-green-500/30",
    },
    python: {
      name: "Python",
      color: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
    },
    rust: {
      name: "Rust",
      color: "bg-orange-500/20 text-orange-400 border-orange-500/30",
    },
    java: {
      name: "Java",
      color: "bg-red-500/20 text-red-400 border-red-500/30",
    },
    ruby: {
      name: "Ruby",
      color: "bg-red-400/20 text-red-300 border-red-400/30",
    },
    php: {
      name: "PHP",
      color: "bg-indigo-500/20 text-indigo-400 border-indigo-500/30",
    },
    dotnet: {
      name: ".NET",
      color: "bg-violet-500/20 text-violet-400 border-violet-500/30",
    },
    elixir: {
      name: "Elixir",
      color: "bg-purple-500/20 text-purple-400 border-purple-500/30",
    },
  };
  return tags[runtime] ?? null;
}

export function detectTechTags(
  appName: string,
  workdir: string,
  repositoryUrl: string,
): readonly TechTag[] {
  const tags: TechTag[] = [];
  const nameAndWorkdir = `${appName} ${workdir} ${repositoryUrl}`.toLowerCase();

  if (
    nameAndWorkdir.includes("go") ||
    nameAndWorkdir.includes("golang") ||
    workdir.includes("cmd/")
  ) {
    tags.push({
      name: "Go",
      color: "bg-cyan-500/20 text-cyan-400 border-cyan-500/30",
    });
  } else if (
    nameAndWorkdir.includes("node") ||
    nameAndWorkdir.includes("express") ||
    nameAndWorkdir.includes("nest")
  ) {
    tags.push({
      name: "Node.js",
      color: "bg-green-500/20 text-green-400 border-green-500/30",
    });
  } else if (
    nameAndWorkdir.includes("python") ||
    nameAndWorkdir.includes("django") ||
    nameAndWorkdir.includes("flask")
  ) {
    tags.push({
      name: "Python",
      color: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
    });
  } else if (nameAndWorkdir.includes("rust")) {
    tags.push({
      name: "Rust",
      color: "bg-orange-500/20 text-orange-400 border-orange-500/30",
    });
  } else if (
    nameAndWorkdir.includes("java") ||
    nameAndWorkdir.includes("spring")
  ) {
    tags.push({
      name: "Java",
      color: "bg-red-500/20 text-red-400 border-red-500/30",
    });
  }

  if (nameAndWorkdir.includes("api")) {
    tags.push({
      name: "API",
      color: "bg-purple-500/20 text-purple-400 border-purple-500/30",
    });
  }

  if (nameAndWorkdir.includes("frontend") || nameAndWorkdir.includes("react")) {
    tags.push({
      name: "Frontend",
      color: "bg-blue-500/20 text-blue-400 border-blue-500/30",
    });
  }

  if (nameAndWorkdir.includes("worker") || nameAndWorkdir.includes("job")) {
    tags.push({
      name: "Worker",
      color: "bg-amber-500/20 text-amber-400 border-amber-500/30",
    });
  }

  return tags;
}
