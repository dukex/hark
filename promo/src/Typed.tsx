export const Typed = ({
  text,
  count,
}: {
  readonly text: string;
  readonly count: number;
}) => (
  <>
    {text.slice(0, Math.max(0, count))}
    <span style={{ opacity: 0 }}>{text.slice(Math.max(0, count))}</span>
  </>
);
