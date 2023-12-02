use tonic::{Request, Response, Status};
use cosmossdk_core::KVStoreService;
use cosmossdk_core::tonic::{context};

pub mod example {
    pub mod counter {
        pub mod v1 {
            include!(concat!(env!("OUT_DIR"), "/example.counter.v1.rs"));
        }
    }
}

struct Counter {
    kv_store_service: KVStoreService,
}

#[tonic::async_trait]
impl example::counter::v1::msg_server::Msg for Counter {
    async fn increment_counter(&self, request: Request<example::counter::v1::IncrementCounterRequest>) -> Result<Response<example::counter::v1::IncrementCounterResponse>, Status> {
        let mut store = self.kv_store_service.open(&mut context(&request));
        if let Some(val) = store.get(&[0]) {
            let mut val_be = u64::from_be_bytes(val[..8].try_into().unwrap());
            val_be += 1;
            store.set(&[0], val_be.to_be_bytes().as_ref());
            Ok(Response::new(example::counter::v1::IncrementCounterResponse {
                current: val_be,
            }))
        } else {
            let val_be = 1u64;
            store.set(&[0], val_be.to_be_bytes().as_ref());
            Ok(Response::new(example::counter::v1::IncrementCounterResponse {
                current: 1,
            }))
        }
    }
}

static PROTO_FILE_DESCRIPTOR_SET: &'static [u8] = &[0u8, 0u8, 0u8, 0u8];

#[no_mangle]
extern fn __proto_file_descriptor_set(len: *mut usize) -> *const u8 {
    unsafe {
        *len = PROTO_FILE_DESCRIPTOR_SET.len();
    }
    PROTO_FILE_DESCRIPTOR_SET.as_ptr()
}