import { IconZoomIn } from "@tabler/icons-react";
import classNames from "classnames";
import { type FC, useState } from "react";
import Modal from "./Modal";

interface ImagePreviewProps {
  src: string;
  alt: string;
  /** Classes for the thumbnail image itself. */
  className?: string;
  /** Shown in the preview's header; falls back to `alt`. */
  caption?: string;
}

// A thumbnail that opens the full-size image in a modal. The trigger is a
// real button so the image is reachable by keyboard, not just by pointer.
const ImagePreview: FC<ImagePreviewProps> = ({
  src,
  alt,
  className,
  caption,
}) => {
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="group relative block w-full overflow-hidden rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:focus-visible:ring-offset-gray-900"
      >
        <img src={src} alt={alt} className={classNames("w-full", className)} />
        <span className="pointer-events-none absolute inset-0 flex items-center justify-center bg-black/0 text-white opacity-0 transition-all duration-150 group-hover:bg-black/40 group-hover:opacity-100">
          <IconZoomIn size={20} />
        </span>
      </button>

      {/* Mounted only while open: Modal keeps its children in the DOM
          regardless of state, and a grid of thumbnails would otherwise leave
          one hidden dialog -- heading and full-size image included -- in the
          accessibility tree per thumbnail. */}
      {open && (
        <Modal
          open={open}
          onClose={() => setOpen(false)}
          className="w-full max-w-4xl"
        >
          <Modal.Header onClose={() => setOpen(false)}>
            <h2 className="truncate text-sm font-semibold text-gray-800 dark:text-gray-100">
              {caption ?? alt}
            </h2>
          </Modal.Header>
          <Modal.Body className="flex justify-center">
            <img
              src={src}
              alt={alt}
              className="max-h-[75vh] w-auto max-w-full rounded-md object-contain"
            />
          </Modal.Body>
        </Modal>
      )}
    </>
  );
};

export default ImagePreview;
