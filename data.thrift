namespace java cn.techwolf.data.gen

// thrift -gen python data.thrift
// thrift -gen java data.thrift

struct Job
{
        1: i32  id,
        2: string title,
        3: string url;
        4: string company,
        5: i32 companyId,

        6: i32 updateDate,
        7: i32 publishDate,
        8: map<string, string> properties;

        9: string desc, // 工作描述 | 工作职责 |
        10: string requirement, // 任职要求

        11: string provider,
        12: string place, // 工作地点

        // 6: string experience, // 工作经验要求
        // 7: string degree, // 学历要求

        13: list<string> jobs, // 职能分类
        14: list<string> industries, // 行业分类
}

struct TinyJob
{
        1: i32 id,
        2: string title,
        3: string company,
        4: string salary,

        5: string place,
        6: i32 companyid,
        7: i32 updateDate, // unix timestamp

        8: optional i32 publishDate,
        9: optional string desc;
        10: optional string url;
}

struct JobProperty
{
    1: i32 id,
    2: i32 experience, // 工作经验
    3: i32 degree,  // 学历要求
    4: i32 updateTs,
    5: i32 publishTs,
//     6: list<i32> jobClass, // 岗位
    7: list<i32> industries, // 行业
    8: i32 companyId, // companyId in kanzhun
    9: i32 salary,
}

struct Jobs
{
    1: map<i32, string> degrees,
    2: map<i32, string> experiences,
    3: map<i32, string> jobClasses,
    4: map<i32, string> industries,
    5: map<i32, string> salaries,
}

enum JobSearchSort {
     DEFAULT = 1,
     ByPubLishTime = 2,
     ByUpdateTime = 3,
}

enum JobCountTimeRange {
     Last1Day = 1,
     Last3Days = 3,
     Last7Days = 7,
}

enum SearchField {
     JobTitle = 1,
     Company = 2,
     FullText = 3,
}

enum FilterField {
     Industry = 1,
     JobClass = 2,
     Salary = 3,
     Degree = 4,
     Experience = 5,
     TimeRange = 6,
}

struct JobSearchReq
{
        1: string query,
        2: string city,
        3: i32 limit,
        4: i32 offset,
        5: SearchField field,

        6: optional map<FilterField, string> filters,
        7: optional string uuid,
        8: optional string uid,
        9: optional i32 debug,
        10: optional bool highlight = true,
}

struct PopularJobReq
{
        1: string city,
        2: i32 limit,
        3: i32 offset,

        4: optional string uuid,
        5: optional string uid,
        6: optional i32 debug,
}

struct CounterItem
{
        1: string name,
        2: i32 count,
        3: i32 id,
}

struct FacetedCounter
{
       1: FilterField name,
       2: list<CounterItem> items,
}

struct JobSearchResp
{
        1: i32 count,
        // 2: list<i32> ids,
        2: list<TinyJob> jobs,
        3: list<FacetedCounter> counters,

        4: i32 milliseconds,

        5: optional map<i32, map<string, string>> debug,
}


struct MobileJobSearchResp {
        1: i32 count,
        2: list<TinyJob> jobs,
        3: i32 milliseconds,

        4: optional list<FacetedCounter> counters,
}

struct AcInput
{
        1: string input,
        2: double score,

        3: string pinyin,
        4: list<i32> pyOffsets,

        5: string initial,
        6: list<i32> initalOffsets,
}

struct Pinyins
{
        1: map<string, string> mapping
}

struct AcRespItem {
    1: required string item,
    3: required double score,
    4: optional string highlighted;
}

struct AcResp
{
    1: required i32 total,
    2: required list<AcRespItem> items,
}

struct AcReq {
    1: required string kind,
    2: required string q,
    3: required i32 limit = 10,

    // track user
    4: optional string uuid,
    5: optional string uid,
    6: optional i32 debug,

    7: optional i32 offset = 0,
    8: optional bool highlight = true,
}

struct JobCountReq {
    1: required string query,
    2: required string city,
    3: required JobCountTimeRange time,
    4: optional string uuid,
    5: optional string uid,

    6: optional i32 count, // 职位数量
}

struct CompanyJobsReq {
       1: string company,
       2: i32 jobid, // current id, result will remove it

       3: i32 limit,
       4: i32 offset,
       5: optional string uuid,
       6: optional string uid,
}


service DataService
{
       JobSearchResp jobSearch(1: JobSearchReq req)
       list<TinyJob> companyJobs(1: CompanyJobsReq req)
       MobileJobSearchResp mobileJobSearch(1: JobSearchReq req)
       MobileJobSearchResp mobilePopular(1: PopularJobReq req)
       list<TinyJob> getJobs(1: list<i32> ids)
       Job getJob(1: i32 id)

       list<i32> jobcount(1: list<JobCountReq> req)
       AcResp autocomplete(1: AcReq req)
       list<string> acnames()
}

service SimilarJobFinderService
{
    list<string> findSimilaryJobs(1: string url)
}


// Liepin: 已抓取职位： 179807 pending： 0; 今日搜索结果页: 8853, pending: 0
// Zhilian: 已抓取职位： 583944 pending： 0; 今日搜索结果页: 15553, pending: 0
// Job51: 已抓取职位： 583944 pending： 0; 今日搜索结果页: 15553, pending: 0
