use anyhow::{bail, Context, Result};
use walrus::{ir::Value, ConstExpr, Data, DataKind, GlobalKind, MemoryId, Module};

pub(crate) fn get_active_data_start(data: &Data, mem: MemoryId) -> Result<u32> {
    let DataKind::Active { memory, offset } = &data.kind else {
        bail!("Adapter data section is not active");
    };
    if *memory != mem {
        bail!("Adapter data memory is not the expected memory id");
    }
    let ConstExpr::Value(Value::I32(loc)) = offset else {
        bail!("Adapter data memory is not absolutely offset");
    };
    Ok(*loc as u32)
}

pub(crate) fn get_active_data_segment(
    module: &mut Module,
    mem: MemoryId,
    addr: u32,
) -> Result<(&mut Data, usize)> {
    let mut found_data: Option<&Data> = None;
    for data in module.data.iter() {
        let data_addr = get_active_data_start(data, mem)?;
        if data_addr <= addr {
            let best_match = match found_data {
                Some(found_data) => data_addr > get_active_data_start(found_data, mem)?,
                None => true,
            };
            if best_match {
                found_data = Some(data);
            }
        }
    }
    let data = found_data.context("Unable to find data section for ptr")?;
    let DataKind::Active {
        offset: ConstExpr::Value(Value::I32(loc)),
        ..
    } = &data.kind
    else {
        unreachable!()
    };
    let loc = *loc as u32;
    let data_id = data.id();
    let offset = (addr - loc) as usize;
    Ok((module.data.get_mut(data_id), offset))
}

pub(crate) fn bump_stack_global(module: &mut Module, offset: i32) -> Result<u32> {
    if offset % 8 != 0 {
        bail!("Stack global must be bumped by 8 byte alignment, offset of {offset} provided");
    }
    let stack_global_id = module
        .globals
        .iter()
        .find(|&global| global.name.as_deref() == Some("__stack_pointer"))
        .context("Unable to find __stack_pointer global name")?
        .id();
    let stack_global = module.globals.get_mut(stack_global_id);
    let GlobalKind::Local(ConstExpr::Value(Value::I32(stack_value))) = &mut stack_global.kind
    else {
        bail!("Stack global is not a constant I32");
    };
    if offset > *stack_value {
        bail!(
            "Stack size {} is smaller than the offset {offset}",
            *stack_value
        );
    }
    let new_stack_value = *stack_value - offset;
    *stack_value = new_stack_value;
    Ok(new_stack_value as u32)
}
