use crate::exports::wasi::config::store::{Error, Guest as Configuration};
use crate::Adapter;

#[repr(C)]
pub struct Config {
    /// [byte 0]
    unused: bool,
    /// How many host fields are defined in the data pointer
    /// [byte 4]
    host_field_cnt: u32,
    /// Byte data of u32 byte len followed by string bytes
    /// up to the lengths previously provided.
    /// [byte 8]
    host_field_data: *const u8,
}

#[no_mangle]
pub static mut CONFIG: Config = Config {
    unused: true,
    host_field_cnt: 0,
    host_field_data: 0 as *const u8,
};

fn read_data_str(offset: &mut isize) -> &'static str {
    let data: *const u8 = unsafe { CONFIG.host_field_data.offset(*offset) };
    let byte_len = unsafe { (data as *const u32).read() } as usize;
    *offset += 4;
    let data: *const u8 = unsafe { CONFIG.host_field_data.offset(*offset) };
    let str_data = unsafe { std::slice::from_raw_parts(data, byte_len) };
    *offset += byte_len as isize;
    let rem = *offset % 4;
    if rem > 0 {
        *offset += 4 - rem;
    }
    unsafe { core::str::from_utf8_unchecked(str_data) }
}

impl Configuration for Adapter {
    fn get(key: String) -> Result<Option<String>, Error> {
        for (k, v) in Self::get_all()? {
            if k == key {
                return Ok(Some(v));
            }
        }

        Ok(None)
    }

    fn get_all() -> Result<Vec<(String, String)>, Error> {
        let mut configuration = Vec::new();
        let mut data_offset: isize = 0;
        for _ in 0..unsafe { CONFIG.host_field_cnt } {
            let config_key = read_data_str(&mut data_offset);
            let config_val = read_data_str(&mut data_offset);
            configuration.push((config_key.to_string(), config_val.to_string()));
        }
        Ok(configuration)
    }
}
