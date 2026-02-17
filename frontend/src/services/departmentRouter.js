import { shivpuriDepartments } from '../data/shivpuriDepartments';

class DepartmentRouter {
  static PRIMARY_EMAIL = 'aineta502@gmail.com';

  static getRecipients(complaint) {
    const { selectedDepartment, location, problem, severity, escalationLevel } = complaint;
    const recipients = new Set([this.PRIMARY_EMAIL]);

    if (selectedDepartment) {
      const dept = shivpuriDepartments.find((d) => d.id === selectedDepartment);
      if (dept && dept.email) {
        recipients.add(dept.email);
      }
    }

    // Add SDM by area (skip Shivpuri city - no SDM for city complaints)
    if (location && location.area && location.area !== 'shivpuri') {
      const areaSdm = shivpuriDepartments.find(
        (d) =>
          d.id === `sdm-${location.area}` ||
          (d.name.includes('एसडीएम') && d.area === location.area)
      );
      if (areaSdm) {
        recipients.add(areaSdm.email);
      }
      // Add Rural Engineering for rural tehsils
      const ruralEngg = shivpuriDepartments.find((d) => d.id === 'rural-engg');
      if (ruralEngg) {
        recipients.add(ruralEngg.email);
      }
    }

    if (severity === 'high' || (escalationLevel && escalationLevel > 1)) {
      const collector = shivpuriDepartments.find((d) => d.id === 'collector');
      if (collector) {
        recipients.add(collector.email);
      }
    }

    const keywordDepartments = this.getDepartmentsByKeywords(problem || '');
    keywordDepartments.forEach((dept) => recipients.add(dept.email));

    return Array.from(recipients);
  }

  static getDepartmentsByKeywords(problem) {
    const keywordMap = [
      { keywords: ['बिजली', 'लाइट', 'वोल्टेज', 'तार', 'खंभा'], deptId: 'electricity' },
      { keywords: ['सड़क', 'रास्ता', 'गड्ढा', 'पुल'], deptId: 'pwd' },
      { keywords: ['पानी', 'नल', 'जल', 'पाइप', 'सप्लाई'], deptId: 'water' },
      { keywords: ['कूड़ा', 'सफाई', 'गंदगी', 'नाली'], deptId: 'municipal' },
      { keywords: ['चोरी', 'लूट', 'मारपीट', 'झगड़ा'], deptId: 'police' },
      { keywords: ['अस्पताल', 'डॉक्टर', 'दवा', 'मरीज'], deptId: 'health' },
      { keywords: ['स्कूल', 'शिक्षक', 'पढ़ाई'], deptId: 'education' }
    ];

    const matchedDepts = [];
    const lowerProblem = (problem || '').toLowerCase();

    keywordMap.forEach((mapping) => {
      if (mapping.keywords.some((keyword) => lowerProblem.includes(keyword))) {
        const dept = shivpuriDepartments.find((d) => d.id === mapping.deptId);
        if (dept) matchedDepts.push(dept);
      }
    });

    return matchedDepts;
  }
}

export default DepartmentRouter;
