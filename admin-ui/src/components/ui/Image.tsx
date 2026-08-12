import { IconPhoto } from "@tabler/icons-react";
import classNames from "classnames";
import { type FC, useRef, useState } from "react";

interface ImageProps {
  src: string;
  alt: string;
  width: number;
  height: number;
  loading?: "lazy" | "eager";
  className?: string;
  containerClassName?: string;
  onLoad?: () => void;
  onError?: () => void;
}

const Image: FC<ImageProps> = ({
  src,
  alt,
  width,
  height,
  loading = "lazy",
  className,
  containerClassName,
  onLoad,
  onError,
}) => {
  const [status, setStatus] = useState<"loading" | "loaded" | "error">(
    "loading",
  );
  const imgRef = useRef<HTMLImageElement>(null);

  const handleLoad = () => {
    setStatus("loaded");
    onLoad?.();
  };

  const handleError = () => {
    setStatus("error");
    onError?.();
  };

  return (
    <div
      className={classNames("relative overflow-hidden", containerClassName)}
      style={{ width, height }}
    >
      {status === "error" ? (
        <div className="flex h-full w-full items-center justify-center rounded-md bg-gray-100 dark:bg-gray-800">
          <div className="flex flex-col items-center gap-1 text-gray-400 dark:text-gray-500">
            <IconPhoto size={24} />
            <span className="text-xs">Failed to load</span>
          </div>
        </div>
      ) : (
        <img
          ref={imgRef}
          src={src}
          alt={alt}
          width={width}
          height={height}
          loading={loading}
          onLoad={handleLoad}
          onError={handleError}
          className={classNames(
            "h-full w-full object-cover transition-opacity duration-300",
            status === "loading" && "opacity-0",
            status === "loaded" && "opacity-100",
            className,
          )}
        />
      )}

      {status === "loading" && (
        <div className="absolute inset-0 animate-pulse rounded-md bg-gray-200 dark:bg-gray-800" />
      )}
    </div>
  );
};

export default Image;
