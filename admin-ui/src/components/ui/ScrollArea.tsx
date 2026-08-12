import classNames from "classnames";
import {
  type FC,
  type ReactNode,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

interface ScrollAreaProps {
  className?: string;
  children: ReactNode;
}

const ScrollArea: FC<ScrollAreaProps> = ({ className, children }) => {
  const [hovered, setHovered] = useState(false);
  const [scrolling, setScrolling] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const [thumbHeight, setThumbHeight] = useState(0);
  const [thumbTop, setThumbTop] = useState(0);

  const updateThumb = useCallback(() => {
    const el = containerRef.current;
    if (!el) {
      return;
    }
    const ratio = el.clientHeight / el.scrollHeight;
    setThumbHeight(Math.max(20, ratio * el.clientHeight));
    setThumbTop((el.scrollTop / el.scrollHeight) * el.clientHeight);
  }, []);

  useEffect(() => {
    updateThumb();
    const el = containerRef.current;
    if (!el) {
      return;
    }
    el.addEventListener("scroll", updateThumb, { passive: true });
    const ro = new ResizeObserver(updateThumb);
    ro.observe(el);
    return () => {
      el.removeEventListener("scroll", updateThumb);
      ro.disconnect();
    };
  }, [updateThumb]);

  let hideTimer: ReturnType<typeof setTimeout>;
  const handleScroll = () => {
    setHovered(true);
    setScrolling(true);
    clearTimeout(hideTimer);
    hideTimer = setTimeout(() => {
      setScrolling(false);
      setHovered(false);
    }, 1200);
  };

  const visible = hovered || scrolling;

  return (
    <div
      ref={containerRef}
      className={classNames("overflow-auto", className)}
      onScroll={handleScroll}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => {
        setHovered(false);
        setScrolling(false);
      }}
    >
      {children}
      <div
        className={classNames(
          "absolute right-0.5 top-0 z-10 w-1.5 rounded-full transition-opacity duration-200",
          visible ? "opacity-100" : "opacity-0",
        )}
        style={{
          height: `${thumbHeight}px`,
          transform: `translateY(${thumbTop}px)`,
        }}
      >
        <div className="h-full w-full rounded-full bg-gray-400/60 dark:bg-gray-500/60" />
      </div>
    </div>
  );
};

export default ScrollArea;
