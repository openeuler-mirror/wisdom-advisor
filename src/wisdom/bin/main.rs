// Copyright (c) 2022 Huawei Technologies Co., Ltd.
// wisdom-advisor is licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Create: 2022-8-6

fn main() {
    let cmd = clap::Command::new("wisdom")
        .bin_name("wisdom")
		.arg(clap::arg!(--loglevel <VALUE> "log level"))
		.arg(clap::Arg::new("printlog").long("printlog").help("output log to terminal for debugging").action(clap::ArgAction::SetTrue))
        .subcommand_required(true)
		.subcommand(
            clap::command!("threadsaffinity")
			.about("trace syscall futex to get thread affinity")
			.arg(clap::arg!(--task <taskname> "the name of the task which needs to be affinity aware").required(true))
			.arg(clap::arg!(--period <period> "scan and balance period").default_value("10"))
			.arg(clap::arg!(--tracetime <tracetime> "time of tracing").default_value("5"))
			.arg(clap::Arg::new("netaware").long("netaware").help("enable net affinity Aware").action(clap::ArgAction::SetTrue))
			.arg(clap::Arg::new("cclaware").long("cclaware").help("bind thread group inside same cluster").action(clap::ArgAction::SetTrue))
			.arg(clap::Arg::new("percore").long("percore").help("bind one thread per core").action(clap::ArgAction::SetTrue))
        )
		.subcommand(
            clap::command!("usersetaffinity")
			.about("parse __SCHED_GROUP__ to get thread affinity")
			.arg(clap::Arg::new("netaware").long("netaware").help("enable net affinity Aware").action(clap::ArgAction::SetTrue))
			.arg(clap::Arg::new("cclaware").long("cclaware").help("bind thread group inside same cluster").action(clap::ArgAction::SetTrue))
			.arg(clap::Arg::new("percore").long("percore").help("bind one thread per core").action(clap::ArgAction::SetTrue))
        )
		.subcommand(
            clap::command!("threadsgrouping")
			.about("trace net and IO syscall, partition threads by user define")
			.arg(clap::arg!(--task <taskname> "the name of the task which needs to be threads grouping").required(true))
			.arg(clap::arg!(--IO <IO> "partition description for IO,like 0-31,64-95"))
			.arg(clap::arg!(--net <net> "partition description for net,like 32-63"))
			.arg(clap::arg!(--period <period> "scan and balance period").default_value("10"))
			.arg(clap::arg!(--tracetime <tracetime> "time of tracing").default_value("5"))
		)
		.subcommand(
            clap::command!("scan")
			.about("thread feature scan control")
			.subcommand(clap::command!("start"))
			.subcommand(clap::command!("stop"))
		);
    let matches = cmd.get_matches();
    let matches = match matches.subcommand() {
        Some(("example", matches)) => matches,
        _ => unreachable!("clap should ensure we don't get here"),
    };
    let manifest_path = matches.get_one::<std::path::PathBuf>("manifest-path");
    println!("{:?}", manifest_path);
}
