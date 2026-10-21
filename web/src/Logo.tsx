import { Image, useComputedColorScheme } from "@mantine/core";

export function Logo({ h }: { h: number }) {
  const scheme = useComputedColorScheme("dark");
  return <Image src={`/api/v1/logo/${scheme}`} h={h} w="auto" fit="contain" alt="" />;
}
