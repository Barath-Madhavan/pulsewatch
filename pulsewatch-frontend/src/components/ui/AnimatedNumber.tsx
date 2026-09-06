import { animate, motion, useMotionValue, useTransform } from "motion/react";
import { useEffect, useRef } from "react";

interface AnimatedNumberProps {
  value: number;
  decimals?: number;
  suffix?: string;
  className?: string;
}

// Counts up (or down) to a new value instead of snapping instantly, so a
// stat that changes on refetch (uptime %, response time) reads as "live"
// rather than a static label that occasionally jumps.
export function AnimatedNumber({ value, decimals = 0, suffix = "", className }: AnimatedNumberProps) {
  const motionValue = useMotionValue(0);
  const display = useTransform(motionValue, (v) => `${v.toFixed(decimals)}${suffix}`);
  const hasMounted = useRef(false);

  useEffect(() => {
    const controls = animate(motionValue, value, {
      duration: hasMounted.current ? 0.6 : 0.9,
      ease: [0.16, 1, 0.3, 1],
    });
    hasMounted.current = true;
    return () => controls.stop();
  }, [value, motionValue]);

  return <motion.span className={className}>{display}</motion.span>;
}
