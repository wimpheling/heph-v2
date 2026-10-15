use connectrpc::{RequestContext, Response, ServiceRequest, ServiceResult};
use heph_proto::heph::system::v1::{
    GetSystemInfoRequest, GetSystemInfoResponse, SystemService as SystemServiceContract,
};

pub struct SystemService;

#[allow(refining_impl_trait)]
impl SystemServiceContract for SystemService {
    async fn get_system_info(
        &self,
        _ctx: RequestContext,
        _request: ServiceRequest<'_, GetSystemInfoRequest>,
    ) -> ServiceResult<GetSystemInfoResponse> {
        let info = heph_core::system_info();
        Response::ok(GetSystemInfoResponse {
            name: info.name.into(),
            version: info.version.into(),
            ..Default::default()
        })
    }
}
