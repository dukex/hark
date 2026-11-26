import "./fonts";
import { Composition } from "remotion";
import { Promo } from "./Promo";

export const RemotionRoot = () => (
  <Composition
    id="Promo"
    component={Promo}
    durationInFrames={1800}
    fps={30}
    width={1920}
    height={1080}
  />
);
