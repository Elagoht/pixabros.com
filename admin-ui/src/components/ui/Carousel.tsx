import { IconChevronLeft, IconChevronRight } from "@tabler/icons-react";
import classNames from "classnames";
import {
  type FC,
  type ReactNode,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

interface CarouselSlide {
  id: string;
  content: ReactNode;
}

interface CarouselProps {
  slides: CarouselSlide[];
  autoPlay?: boolean;
  interval?: number;
  showDots?: boolean;
  showArrows?: boolean;
  aspectRatio?: string;
  className?: string;
}

const Carousel: FC<CarouselProps> = ({
  slides,
  autoPlay = false,
  interval = 4000,
  showDots = true,
  showArrows = true,
  aspectRatio,
  className,
}) => {
  const [current, setCurrent] = useState(0);
  const [dragOffset, setDragOffset] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const timerRef = useRef<ReturnType<typeof setInterval> | undefined>(
    undefined,
  );
  const containerRef = useRef<HTMLDivElement>(null);
  const dragStartX = useRef(0);
  const dragCurrentX = useRef(0);

  const goTo = useCallback(
    (index: number) => {
      setCurrent((index + slides.length) % slides.length);
    },
    [slides.length],
  );

  const next = useCallback(() => goTo(current + 1), [current, goTo]);
  const prev = useCallback(() => goTo(current - 1), [current, goTo]);

  useEffect(() => {
    if (!autoPlay) {
      return;
    }
    timerRef.current = setInterval(() => {
      setCurrent((c) => (c + 1) % slides.length);
    }, interval);
    return () => clearInterval(timerRef.current);
  }, [autoPlay, interval, slides.length]);

  const handlePointerDown = useCallback(
    (e: React.PointerEvent) => {
      if (slides.length <= 1) {
        return;
      }
      if (
        (e.target as HTMLElement).closest("button, a, input, select, textarea")
      ) {
        return;
      }
      setIsDragging(true);
      dragStartX.current = e.clientX;
      dragCurrentX.current = e.clientX;
      setDragOffset(0);
      if (containerRef.current) {
        containerRef.current.setPointerCapture(e.pointerId);
      }
    },
    [slides.length],
  );

  const handlePointerMove = useCallback(
    (e: React.PointerEvent) => {
      if (!isDragging) {
        return;
      }
      dragCurrentX.current = e.clientX;
      setDragOffset(dragCurrentX.current - dragStartX.current);
    },
    [isDragging],
  );

  const handlePointerUp = useCallback(
    (e: React.PointerEvent) => {
      if (!isDragging) {
        return;
      }
      setIsDragging(false);

      if (containerRef.current) {
        containerRef.current.releasePointerCapture(e.pointerId);
      }

      const delta = dragCurrentX.current - dragStartX.current;
      const width = containerRef.current?.offsetWidth ?? 0;
      const threshold = width * 0.2;

      if (Math.abs(delta) > threshold) {
        if (delta > 0) {
          goTo(current - 1);
        } else {
          goTo(current + 1);
        }
      }

      setDragOffset(0);
    },
    [isDragging, current, goTo],
  );

  if (slides.length === 0) {
    return null;
  }

  const translateX = `calc(-${current * 100}% + ${dragOffset}px)`;

  return (
    <div
      ref={containerRef}
      className={classNames(
        "relative select-none overflow-hidden rounded-lg",
        isDragging ? "cursor-grabbing" : "cursor-grab",
        className,
      )}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      onPointerCancel={handlePointerUp}
      onMouseEnter={() => {
        if (autoPlay) {
          clearInterval(timerRef.current);
        }
      }}
      onMouseLeave={() => {
        if (autoPlay) {
          timerRef.current = setInterval(() => {
            setCurrent((c) => (c + 1) % slides.length);
          }, interval);
        }
      }}
    >
      <div
        className={classNames(
          "flex",
          !isDragging && "transition-transform duration-500 ease-out",
        )}
        style={{ transform: `translateX(${translateX})` }}
      >
        {slides.map((slide) => (
          <div
            key={slide.id}
            className="w-full shrink-0"
            style={aspectRatio ? { aspectRatio } : undefined}
          >
            {slide.content}
          </div>
        ))}
      </div>

      {showArrows && slides.length > 1 && (
        <>
          <button
            type="button"
            onPointerDown={(e) => e.stopPropagation()}
            onClick={prev}
            className="absolute left-2 top-1/2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full bg-white/80 text-gray-700 shadow transition-colors hover:bg-white dark:bg-gray-900/80 dark:text-gray-300 dark:hover:bg-gray-900"
          >
            <IconChevronLeft size={18} />
          </button>
          <button
            type="button"
            onPointerDown={(e) => e.stopPropagation()}
            onClick={next}
            className="absolute right-2 top-1/2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full bg-white/80 text-gray-700 shadow transition-colors hover:bg-white dark:bg-gray-900/80 dark:text-gray-300 dark:hover:bg-gray-900"
          >
            <IconChevronRight size={18} />
          </button>
        </>
      )}

      {showDots && slides.length > 1 && (
        <div className="absolute bottom-3 left-1/2 flex -translate-x-1/2 items-center gap-2">
          {slides.map((_, i) => (
            <button
              key={i}
              type="button"
              onPointerDown={(e) => e.stopPropagation()}
              onClick={() => setCurrent(i)}
              className={classNames(
                "h-2 rounded-full transition-all",
                i === current
                  ? "w-6 bg-white shadow"
                  : "w-2 bg-white/60 hover:bg-white/80",
              )}
            />
          ))}
        </div>
      )}
    </div>
  );
};

export default Carousel;
