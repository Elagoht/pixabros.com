import { IconX, IconZoomIn } from "@tabler/icons-react";
import classNames from "classnames";
import { type FC, useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useI18n } from "@/lib/stores/i18n";

interface ImagePreviewProps {
  src: string;
  alt: string;
  /** Classes for the thumbnail image itself. */
  className?: string;
}

/** Kept in step with the transition durations below. */
const TRANSITION_MS = 200;

// A thumbnail that opens the image full-size in a lightbox: a dim overlay and
// nothing else, so the picture gets the whole viewport rather than being
// boxed inside a dialog panel. The trigger is a real button so the image is
// reachable by keyboard, not just by pointer.
const ImagePreview: FC<ImagePreviewProps> = ({ src, alt, className }) => {
  const { t } = useI18n();
  // Two flags rather than one: `mounted` decides whether the lightbox is in
  // the DOM at all, `shown` drives the transition. Opening mounts first and
  // reveals on the next frame -- a class changed in the same frame as the
  // insertion would jump straight to the end state instead of animating.
  // Closing reverses it, unmounting only once the exit transition has run.
  const [mounted, setMounted] = useState(false);
  const [shown, setShown] = useState(false);
  const exitTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const open = useCallback(() => {
    if (exitTimer.current) {
      clearTimeout(exitTimer.current);
      exitTimer.current = null;
    }
    setMounted(true);
  }, []);

  const close = useCallback(() => {
    setShown(false);
    exitTimer.current = setTimeout(() => {
      setMounted(false);
      exitTimer.current = null;
    }, TRANSITION_MS);
  }, []);

  useEffect(() => {
    if (!mounted) {
      return;
    }
    const frame = requestAnimationFrame(() => setShown(true));
    return () => cancelAnimationFrame(frame);
  }, [mounted]);

  // A pending exit timer must not fire after the component has gone.
  useEffect(
    () => () => {
      if (exitTimer.current) {
        clearTimeout(exitTimer.current);
      }
    },
    [],
  );

  useEffect(() => {
    if (!mounted) {
      return;
    }
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        close();
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [mounted, close]);

  useEffect(() => {
    if (!mounted) {
      return;
    }
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previous;
    };
  }, [mounted]);

  return (
    <>
      <button
        type="button"
        onClick={open}
        className="group relative block w-full overflow-hidden rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:focus-visible:ring-offset-gray-900"
      >
        <img src={src} alt={alt} className={classNames("w-full", className)} />
        <span className="pointer-events-none absolute inset-0 flex items-center justify-center bg-black/0 text-white opacity-0 transition-all duration-150 group-hover:bg-black/40 group-hover:opacity-100">
          <IconZoomIn size={20} />
        </span>
      </button>

      {/* Mounted only while open: a grid of thumbnails would otherwise leave
          one hidden full-size image per thumbnail in the DOM. */}
      {mounted &&
        createPortal(
          <div
            role="dialog"
            aria-modal="true"
            aria-label={alt}
            onClick={close}
            className={classNames(
              "fixed inset-0 z-50 flex items-center justify-center bg-black/80",
              "transition-opacity duration-200 ease-out",
              shown ? "opacity-100" : "opacity-0",
            )}
          >
            <button
              type="button"
              onClick={close}
              aria-label={t("common.close")}
              className="absolute right-4 top-4 z-10 rounded-full bg-white/10 p-2 text-white transition-colors hover:bg-white/20 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/60"
            >
              <IconX size={20} />
            </button>

            {/* Clicking the picture itself must not dismiss it. */}
            <img
              src={src}
              alt={alt}
              onClick={(e) => e.stopPropagation()}
              className={classNames(
                "max-h-[90vh] max-w-[90vw] object-contain",
                "transition-all duration-200 ease-out",
                shown ? "scale-100 opacity-100" : "scale-95 opacity-0",
              )}
            />
          </div>,
          document.body,
        )}
    </>
  );
};

export default ImagePreview;
