// 后台接口统一返回 envelope；前端通过这层类型拿到真正的数据载荷。
export interface ApiEnvelope<T> {
  code: number;
  msg: string;
  uuid: string;
  data: T;
}
