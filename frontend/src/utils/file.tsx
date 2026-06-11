export const getFileName = (path: string) => {
  return path.split('/').pop() || path;
};

export const isImagePath = (path: string | null | undefined) => {
  if (!path) return false;
  return /\.(jpg|jpeg|png|gif|webp)$/i.test(path);
};
