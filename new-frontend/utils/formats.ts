export const formatDuration = (inputMs: number) => {
  let ms = inputMs;
  if (ms < 0) {
    ms = -ms;
  }

  ms /= 1_000;

  const time = {
    d: Math.floor(ms / 86_400_000),
    h: Math.floor(ms / 3_600_000) % 24,
    m: Math.floor(ms / 60_000) % 60,
    s: Math.floor(ms / 1_000) % 60,
    ms: ms % 1_000,
  };

  return Object.entries(time)
    .filter((val) => val[1] !== 0)
    .map((val) => `${val[1].toFixed(4)} ${val[1] !== 1 ? val[0] : val[0]}`)
    .join(", ");
}

export const humanFileSize = (inputBytes: number) => {
  let bytes = inputBytes;
  const thresh = 1024;

  if (Math.abs(bytes) < thresh) {
    return `${bytes} B`;
  }

  const units = ["KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"];
  let u = -1;
  const r = 10 ** 1;

  do {
    bytes /= thresh;
    u += 1;
  } while (
    Math.round(Math.abs(bytes) * r) / r >= thresh &&
    u < units.length - 1
    );

  return `${bytes.toFixed(1)} ${units[u]}`;
}
