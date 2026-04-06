import { motion } from "motion/react";

interface BackgroundLinesProps {
  /** The main color for the lines - accepts any valid CSS color */
  color?: string;
  /** Number of lines to render (6-8 recommended) */
  lineCount?: number;
  /** Animation duration in seconds */
  duration?: number;
  /** Additional className for the container */
  className?: string;
  /** Children to render on top of the background */
  children?: React.ReactNode;
}

export function BackgroundLines({
  color = "#006045",
  lineCount = 7,
  duration = 8,
  className = "",
  children,
}: BackgroundLinesProps) {
  // Generate path from true top-left corner to true bottom-right corner with swirl in middle
  const generatePath = (index: number, total: number) => {
    // Spread lines across a range at the start and end
    const spread = 25;
    const offset = ((index - total / 2) / total) * spread;
    
    // Swirl variations per line
    const swirlOffset = Math.sin(index * 0.9) * 8;
    const swirlIntensity = 15 + Math.cos(index * 0.7) * 5;

    // Start at top-left, swirl through center, end at bottom-right
    // Using viewBox coordinates: 0,0 is top-left, 100,100 is bottom-right
    const startX = -5 + offset * 0.5;
    const startY = -5 + offset;
    const endX = 105 + offset * 0.5;
    const endY = 105 + offset;
    
    // Control points for the swirl in the center
    const cp1x = 25 + swirlOffset;
    const cp1y = 30 + offset + swirlIntensity;
    const midX = 50 + swirlOffset * 0.5;
    const midY = 50 + swirlOffset;
    const cp2x = 75 - swirlOffset;
    const cp2y = 70 + offset - swirlIntensity;

    return `M ${startX} ${startY}
            Q ${cp1x} ${cp1y}, ${midX} ${midY}
            Q ${cp2x} ${cp2y}, ${endX} ${endY}`;
  };

  return (
    <div
      className={`relative w-full h-full overflow-hidden bg-neutral-950 ${className}`}
    >
      {/* SVG Background */}
      <svg
        className="absolute inset-0 w-full h-full"
        viewBox="0 0 100 100"
        preserveAspectRatio="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        <defs>
          {/* Enhanced glow filter */}
          <filter id="glow" x="-50%" y="-50%" width="200%" height="200%">
            <feGaussianBlur stdDeviation="0.3" result="blur1" />
            <feGaussianBlur stdDeviation="0.8" result="blur2" />
            <feGaussianBlur stdDeviation="1.5" result="blur3" />
            <feMerge>
              <feMergeNode in="blur3" />
              <feMergeNode in="blur2" />
              <feMergeNode in="blur1" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>

          {/* Intense outer glow */}
          <filter id="glowIntense" x="-100%" y="-100%" width="300%" height="300%">
            <feGaussianBlur stdDeviation="1" result="blur1" />
            <feGaussianBlur stdDeviation="2.5" result="blur2" />
            <feGaussianBlur stdDeviation="5" result="blur3" />
            <feMerge>
              <feMergeNode in="blur3" />
              <feMergeNode in="blur2" />
              <feMergeNode in="blur1" />
            </feMerge>
          </filter>

          {/* Gradient for line fade at edges */}
          <linearGradient id="lineGradient" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor={color} stopOpacity="0" />
            <stop offset="10%" stopColor={color} stopOpacity="0.8" />
            <stop offset="50%" stopColor={color} stopOpacity="1" />
            <stop offset="90%" stopColor={color} stopOpacity="0.8" />
            <stop offset="100%" stopColor={color} stopOpacity="0" />
          </linearGradient>
        </defs>

        {/* Outer glow layer */}
        <g filter="url(#glowIntense)" style={{ mixBlendMode: "screen" }}>
          {Array.from({ length: lineCount }).map((_, i) => (
            <motion.path
              key={`outer-glow-${i}`}
              d={generatePath(i, lineCount)}
              fill="none"
              stroke={color}
              strokeWidth={0.6}
              strokeOpacity={0.3}
              strokeLinecap="round"
              initial={{
                pathLength: 0,
                pathOffset: 0,
              }}
              animate={{
                pathLength: [0, 1, 1],
                pathOffset: [0, 0, 1],
              }}
              transition={{
                duration: duration + i * 0.4,
                delay: i * 0.3,
                repeat: Infinity,
                ease: "easeInOut",
                times: [0, 0.5, 1],
              }}
            />
          ))}
        </g>

        {/* Main lines with glow */}
        <g filter="url(#glow)">
          {Array.from({ length: lineCount }).map((_, i) => {
            const opacity = 0.7 + Math.sin(i * 0.6) * 0.25;

            return (
              <motion.path
                key={`main-${i}`}
                d={generatePath(i, lineCount)}
                fill="none"
                stroke="url(#lineGradient)"
                strokeWidth={0.15 + Math.sin(i * 0.5) * 0.05}
                strokeOpacity={opacity}
                strokeLinecap="round"
                initial={{
                  pathLength: 0,
                  pathOffset: 0,
                }}
                animate={{
                  pathLength: [0, 1, 1],
                  pathOffset: [0, 0, 1],
                }}
                transition={{
                  duration: duration + i * 0.3,
                  delay: i * 0.35,
                  repeat: Infinity,
                  ease: "easeInOut",
                  times: [0, 0.5, 1],
                }}
              />
            );
          })}
        </g>

        {/* Bright core highlight */}
        <g filter="url(#glow)" style={{ mixBlendMode: "screen" }}>
          {Array.from({ length: lineCount }).map((_, i) => (
            <motion.path
              key={`core-${i}`}
              d={generatePath(i, lineCount)}
              fill="none"
              stroke={color}
              strokeWidth={0.4}
              strokeOpacity={0.2}
              strokeLinecap="round"
              initial={{
                pathLength: 0,
                pathOffset: 0,
              }}
              animate={{
                pathLength: [0, 1, 1],
                pathOffset: [0, 0, 1],
              }}
              transition={{
                duration: duration + i * 0.3,
                delay: i * 0.35 + 0.05,
                repeat: Infinity,
                ease: "easeInOut",
                times: [0, 0.5, 1],
              }}
            />
          ))}
        </g>
      </svg>

      {/* Content layer */}
      {children && (
        <div className="relative z-10 w-full h-full flex items-center justify-center">
          {children}
        </div>
      )}
    </div>
  );
}
