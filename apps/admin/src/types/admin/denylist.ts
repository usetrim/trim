export type DenylistEmailBody = {
  domain: string;
  reason: string;
};

export type DenylistIpBody = {
  cidr: string;
  reason: string;
};

export type DenylistAsnBody = {
  asn: number;
  reason: string;
};
